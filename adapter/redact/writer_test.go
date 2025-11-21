package redact

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockWriteCloser is a mock writer that tracks writes and close calls
type mockWriteCloser struct {
	buf    *bytes.Buffer
	closed bool
	mu     sync.Mutex
}

func newMockWriteCloser() *mockWriteCloser {
	return &mockWriteCloser{
		buf: &bytes.Buffer{},
	}
}

func (m *mockWriteCloser) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.Write(p)
}

func (m *mockWriteCloser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockWriteCloser) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.String()
}

func (m *mockWriteCloser) WasClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func TestRedactingWriter_BasicRedaction(t *testing.T) {
	tests := []struct {
		name     string
		secrets  []string
		input    string
		expected string
	}{
		{
			name:     "single secret in single write",
			secrets:  []string{"password123"},
			input:    "user password123 logged in",
			expected: "user ******* logged in",
		},
		{
			name:     "multiple secrets",
			secrets:  []string{"password123", "secret-key"},
			input:    "user password123 with secret-key access",
			expected: "user ******* with ******* access",
		},
		{
			name:     "no secrets to redact",
			secrets:  []string{"password123"},
			input:    "normal log message",
			expected: "normal log message",
		},
		{
			name:     "empty input",
			secrets:  []string{"password123"},
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(tt.secrets...)
			mock := newMockWriteCloser()
			writer := NewRedactingWriter(mock, store)

			_, err := writer.Write([]byte(tt.input))
			require.NoError(t, err)

			// close to flush buffer
			err = writer.Close()
			require.NoError(t, err)

			require.Equal(t, tt.expected, mock.String())
		})
	}
}

func TestRedactingWriter_SplitSecretAcrossWrites(t *testing.T) {
	tests := []struct {
		name     string
		secrets  []string
		writes   []string
		expected string
	}{
		{
			name:     "secret split in middle",
			secrets:  []string{"password123"},
			writes:   []string{"user pass", "word123 logged in"},
			expected: "user ******* logged in",
		},
		{
			name:     "secret split at different positions",
			secrets:  []string{"api-key-12345"},
			writes:   []string{"request with api-", "key-12345 sent"},
			expected: "request with ******* sent",
		},
		{
			name:     "secret across three writes",
			secrets:  []string{"longsecret"},
			writes:   []string{"prefix long", "sec", "ret suffix"},
			expected: "prefix ******* suffix",
		},
		{
			name:     "multiple secrets split differently",
			secrets:  []string{"secret1", "secret2"},
			writes:   []string{"has sec", "ret1 and secr", "et2 here"},
			expected: "has ******* and ******* here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(tt.secrets...)
			mock := newMockWriteCloser()
			writer := NewRedactingWriter(mock, store)

			for _, write := range tt.writes {
				_, err := writer.Write([]byte(write))
				require.NoError(t, err)
			}

			// close to flush buffer
			err := writer.Close()
			require.NoError(t, err)

			require.Equal(t, tt.expected, mock.String())
		})
	}
}

func TestRedactingWriter_LargeWrite(t *testing.T) {
	secret := "verylongsecretkey12345"
	store := NewStore(secret)
	mock := newMockWriteCloser()
	writer := NewRedactingWriter(mock, store)

	// create a large write that exceeds the buffer window
	largeInput := "start " + secret + " middle "
	for i := 0; i < 100; i++ {
		largeInput += "some normal text "
	}
	largeInput += secret + " end"

	_, err := writer.Write([]byte(largeInput))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	output := mock.String()
	require.NotContains(t, output, secret, "secret should be redacted")
	require.Contains(t, output, "*******", "redaction marker should be present")
}

func TestRedactingWriter_Close(t *testing.T) {
	t.Run("flushes remaining buffer", func(t *testing.T) {
		store := NewStore("secret")
		mock := newMockWriteCloser()
		writer := NewRedactingWriter(mock, store)

		// write data that's smaller than buffer window (window = 2 * 6 = 12 for "secret")
		_, err := writer.Write([]byte("my secret"))
		require.NoError(t, err)

		// at this point, data should still be in buffer (9 chars < 12 char window)
		require.Empty(t, mock.String())

		// close should flush the buffer
		err = writer.Close()
		require.NoError(t, err)

		require.Equal(t, "my *******", mock.String())
	})

	t.Run("closes underlying closeable writer", func(t *testing.T) {
		store := NewStore("secret")
		mock := newMockWriteCloser()
		writer := NewRedactingWriter(mock, store)

		err := writer.Close()
		require.NoError(t, err)

		require.True(t, mock.WasClosed(), "underlying writer should be closed")
	})

	t.Run("handles non-closeable writer", func(t *testing.T) {
		store := NewStore("secret")
		buf := &bytes.Buffer{} // bytes.Buffer doesn't implement Close
		writer := NewRedactingWriter(buf, store)

		// should not panic or error
		err := writer.Close()
		require.NoError(t, err)
	})
}

func TestRedactingWriter_ConcurrentWrites(t *testing.T) {
	store := NewStore("secret1", "secret2")
	mock := newMockWriteCloser()
	writer := NewRedactingWriter(mock, store)

	// write concurrently from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				_, err := writer.Write([]byte("goroutine has secret1 and secret2 \n"))
				require.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	err := writer.Close()
	require.NoError(t, err)

	output := mock.String()
	require.NotContains(t, output, "secret1", "secret1 should be redacted")
	require.NotContains(t, output, "secret2", "secret2 should be redacted")
}

func TestRedactingWriter_RedactorCollection(t *testing.T) {
	store1 := NewStore("password")
	store2 := NewStore("apikey")
	collection := newRedactorCollection(store1, store2)

	mock := newMockWriteCloser()
	writer := NewRedactingWriter(mock, collection)

	_, err := writer.Write([]byte("user password with apikey access"))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	require.Equal(t, "user ******* with ******* access", mock.String())
}

func TestRedactingWriter_DynamicSecretAddition(t *testing.T) {
	store := NewStore("initial")
	mock := newMockWriteCloser()
	writer := NewRedactingWriter(mock, store)

	// write with initial secret
	_, err := writer.Write([]byte("has initial "))
	require.NoError(t, err)

	// add new secret dynamically
	if sw, ok := store.(StoreWriter); ok {
		sw.Add("newsecret")
	}

	// write with both secrets
	_, err = writer.Write([]byte("and newsecret here"))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	output := mock.String()
	require.NotContains(t, output, "initial", "initial secret should be redacted")
	require.NotContains(t, output, "newsecret", "new secret should be redacted")
}

func TestRedactingWriter_BufferWindowSize(t *testing.T) {
	// test that the buffer window is indeed 2x the longest secret
	secret := "verylongsecretvalue"
	store := NewStore(secret)
	mock := newMockWriteCloser()
	writer := NewRedactingWriter(mock, store).(*redactingWriter)

	maxLen := writer.maxSecretLength()
	require.Equal(t, len(secret), maxLen, "max secret length should match the secret")

	// write data smaller than window size (2x secret length)
	smallData := make([]byte, len(secret))
	_, err := writer.Write(smallData)
	require.NoError(t, err)

	// data should still be in buffer (not flushed yet)
	require.Empty(t, mock.String(), "data smaller than window should remain buffered")

	// write data that exceeds window size
	largeData := make([]byte, len(secret)*2)
	_, err = writer.Write(largeData)
	require.NoError(t, err)

	// some data should have been flushed
	require.NotEmpty(t, mock.String(), "data exceeding window should be flushed")
}

func TestRedactingWriter_EmptyStore(t *testing.T) {
	// test with a store that has no secrets
	store := NewStore()
	mock := newMockWriteCloser()
	writer := NewRedactingWriter(mock, store)

	input := "normal message with no secrets"
	_, err := writer.Write([]byte(input))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	require.Equal(t, input, mock.String(), "output should match input when no secrets are defined")
}
