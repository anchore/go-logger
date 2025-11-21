package redact

import (
	"io"
	"strings"
	"sync"
)

// redactingWriter wraps an io.Writer and redacts secrets before writing to the underlying writer.
// it maintains a sliding window buffer to catch secrets that may be split across Write() calls.
type redactingWriter struct {
	underlying io.Writer
	redactor   Redactor
	buffer     []byte
	lock       sync.Mutex
}

var _ io.WriteCloser = (*redactingWriter)(nil)

// NewRedactingWriter creates a new io.WriteCloser that wraps the given writer and applies
// redaction using the provided Redactor. The writer maintains a sliding window buffer to
// catch secrets that may be split across multiple Write() calls.
func NewRedactingWriter(w io.Writer, r Redactor) io.WriteCloser {
	return &redactingWriter{
		underlying: w,
		redactor:   r,
		buffer:     make([]byte, 0),
	}
}

// maxSecretLength returns the length of the longest secret tracked by the redactor.
// this is used to determine the sliding window buffer size (2x this value).
func (w *redactingWriter) maxSecretLength() int {
	values := w.getRedactorValues()
	if len(values) == 0 {
		// default minimum buffer size if no secrets are present
		return 64
	}

	maxLen := 0
	for _, v := range values {
		if len(v) > maxLen {
			maxLen = len(v)
		}
	}
	return maxLen
}

// getRedactorValues extracts all redaction values from the redactor.
// it handles both store and redactorCollection types using type assertions.
func (w *redactingWriter) getRedactorValues() []string {
	switch r := w.redactor.(type) {
	case *store:
		return r.values()
	case redactorCollection:
		var allValues []string
		for _, redactor := range r {
			// recursively create a temporary writer to get values
			tempWriter := &redactingWriter{redactor: redactor}
			allValues = append(allValues, tempWriter.getRedactorValues()...)
		}
		return allValues
	default:
		// for unknown redactor types, we can't determine the values
		return nil
	}
}

// Write implements io.Writer, buffering data and applying redaction before writing to the underlying writer.
// it maintains a sliding window buffer (2x the longest secret length) to catch secrets that may be
// split across multiple Write() calls. When the buffer exceeds the window size, the excess is redacted
// and written to the underlying writer.
//
// Note: To properly handle secrets that may span the flush boundary, we redact the entire buffer
// before flushing. This ensures secrets are never partially written. The window is maintained in
// redacted form to prevent keeping remnants of already-flushed secrets.
func (w *redactingWriter) Write(p []byte) (n int, err error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	// append incoming data to buffer
	w.buffer = append(w.buffer, p...)

	windowSize := 2 * w.maxSecretLength()

	// if buffer exceeds window size, flush the excess
	if len(w.buffer) > windowSize {
		// redact the entire buffer to properly handle secrets spanning the flush boundary
		redactedFull := w.redactor.RedactString(string(w.buffer))

		// calculate flush point in original buffer
		origFlushLen := len(w.buffer) - windowSize

		// map the flush point from original to redacted coordinates
		redactedFlushLen := w.mapPosition(string(w.buffer), redactedFull, origFlushLen)

		// write the redacted portion
		_, err = w.underlying.Write([]byte(redactedFull[:redactedFlushLen]))
		if err != nil {
			return len(p), err
		}

		// keep the redacted window (not original) to maintain consistency
		// this prevents keeping remnants of secrets that were already redacted and flushed
		w.buffer = []byte(redactedFull[redactedFlushLen:])
	}

	return len(p), nil
}

// mapPosition maps a position in the original string to the corresponding position
// in the redacted string, accounting for secrets being replaced with fixed-length markers.
func (w *redactingWriter) mapPosition(original, redacted string, origPos int) int {
	if origPos >= len(original) {
		return len(redacted)
	}

	// scan both strings in parallel, tracking positions
	oPos, rPos := 0, 0
	values := w.getRedactorValues()
	redactionMarker := strings.Repeat("*", 7)

	for oPos < origPos && oPos < len(original) {
		// check if current position in original starts with any secret
		matched := false
		for _, secret := range values {
			if oPos+len(secret) <= len(original) && original[oPos:oPos+len(secret)] == secret {
				// found a secret, skip it in original and skip the marker in redacted
				oPos += len(secret)
				rPos += len(redactionMarker)
				matched = true
				break
			}
		}

		if !matched {
			// no secret, advance both by one character
			oPos++
			rPos++
		}
	}

	return rPos
}

// Close implements io.Closer, flushing any remaining buffered data (after redaction) and
// closing the underlying writer if it implements io.Closer.
func (w *redactingWriter) Close() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	// redact and flush any remaining buffered data
	if len(w.buffer) > 0 {
		redacted := w.redactor.RedactString(string(w.buffer))
		_, err := w.underlying.Write([]byte(redacted))
		if err != nil {
			return err
		}
		w.buffer = nil
	}

	// close the underlying writer if it implements io.Closer
	if closer, ok := w.underlying.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}
