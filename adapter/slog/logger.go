// Package slog provides a forward adapter that exposes an iface.Logger backed
// by a standard library *log/slog.Logger.
package slog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"math"
	"os"

	iface "github.com/anchore/go-logger"
)

// Format selects the slog handler used when constructing a logger via New.
type Format int

const (
	// FormatText uses slog.NewTextHandler.
	FormatText Format = iota
	// FormatJSON uses slog.NewJSONHandler.
	FormatJSON
)

// LevelTrace is the slog level used by this adapter to represent
// iface.TraceLevel. It is one step below slog.LevelDebug. Callers building
// their own *slog.Logger for use with Use must configure their handler's
// minimum level at or below LevelTrace if they want trace logging emitted.
const LevelTrace slog.Level = slog.LevelDebug - 4

// Config contains all configurable values for building a slog-backed logger.
type Config struct {
	Level     iface.Level
	Output    io.Writer
	Format    Format
	AddSource bool
}

func DefaultConfig() Config {
	return Config{
		Level:  iface.InfoLevel,
		Output: os.Stderr,
		Format: FormatText,
	}
}

var (
	_ iface.Logger     = (*logger)(nil)
	_ iface.Controller = (*logger)(nil)
	_ iface.Logger     = (*wrappedLogger)(nil)
)

// logger backs a slog.Logger that this adapter constructed via New, so we
// retain enough configuration to rebuild the handler when SetOutput is called.
type logger struct {
	cfg    Config
	output io.Writer
	slog   *slog.Logger
}

// wrappedLogger backs a slog.Logger supplied by the caller via Use. We treat
// the caller's handler as opaque: we cannot rebuild it on SetOutput, so this
// type does not implement iface.Controller.
type wrappedLogger struct {
	slog *slog.Logger
}

// New builds a *slog.Logger from cfg and adapts it.
func New(cfg Config) iface.Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stderr
	}
	return &logger{
		cfg:    cfg,
		output: cfg.Output,
		slog:   slog.New(buildHandler(cfg.Output, cfg)),
	}
}

// Use wraps an already-configured *slog.Logger. The caller's handler, level,
// output, and source-info settings are taken as-is; cfg is not re-applied.
// Callers are responsible for ensuring their handler permits LevelTrace if
// trace logging is desired.
func Use(l *slog.Logger) iface.Logger {
	return &wrappedLogger{slog: l}
}

func buildHandler(w io.Writer, cfg Config) slog.Handler {
	opts := &slog.HandlerOptions{
		Level:     translateLevel(cfg.Level),
		AddSource: cfg.AddSource,
	}
	switch cfg.Format {
	case FormatJSON:
		return slog.NewJSONHandler(w, opts)
	default:
		return slog.NewTextHandler(w, opts)
	}
}

func translateLevel(l iface.Level) slog.Level {
	switch l {
	case iface.TraceLevel:
		return LevelTrace
	case iface.DebugLevel:
		return slog.LevelDebug
	case iface.InfoLevel:
		return slog.LevelInfo
	case iface.WarnLevel:
		return slog.LevelWarn
	case iface.ErrorLevel:
		return slog.LevelError
	case iface.DisabledLevel:
		return slog.Level(math.MaxInt)
	}
	return slog.LevelInfo
}

func (l *logger) Tracef(format string, args ...any) { logf(l.slog, LevelTrace, format, args...) }
func (l *logger) Debugf(format string, args ...any) { logf(l.slog, slog.LevelDebug, format, args...) }
func (l *logger) Infof(format string, args ...any)  { logf(l.slog, slog.LevelInfo, format, args...) }
func (l *logger) Warnf(format string, args ...any)  { logf(l.slog, slog.LevelWarn, format, args...) }
func (l *logger) Errorf(format string, args ...any) { logf(l.slog, slog.LevelError, format, args...) }

func (l *logger) Trace(args ...any) { logArgs(l.slog, LevelTrace, args...) }
func (l *logger) Debug(args ...any) { logArgs(l.slog, slog.LevelDebug, args...) }
func (l *logger) Info(args ...any)  { logArgs(l.slog, slog.LevelInfo, args...) }
func (l *logger) Warn(args ...any)  { logArgs(l.slog, slog.LevelWarn, args...) }
func (l *logger) Error(args ...any) { logArgs(l.slog, slog.LevelError, args...) }

func (l *logger) WithFields(fields ...any) iface.MessageLogger {
	return &wrappedLogger{slog: l.slog.With(getFields(fields...)...)}
}

func (l *logger) Nested(fields ...any) iface.Logger {
	return &wrappedLogger{slog: l.slog.With(getFields(fields...)...)}
}

func (l *logger) SetOutput(w io.Writer) {
	l.output = w
	l.slog = slog.New(buildHandler(w, l.cfg))
}

func (l *logger) GetOutput() io.Writer { return l.output }

func (w *wrappedLogger) Tracef(format string, args ...any) {
	logf(w.slog, LevelTrace, format, args...)
}
func (w *wrappedLogger) Debugf(format string, args ...any) {
	logf(w.slog, slog.LevelDebug, format, args...)
}
func (w *wrappedLogger) Infof(format string, args ...any) {
	logf(w.slog, slog.LevelInfo, format, args...)
}
func (w *wrappedLogger) Warnf(format string, args ...any) {
	logf(w.slog, slog.LevelWarn, format, args...)
}
func (w *wrappedLogger) Errorf(format string, args ...any) {
	logf(w.slog, slog.LevelError, format, args...)
}

func (w *wrappedLogger) Trace(args ...any) { logArgs(w.slog, LevelTrace, args...) }
func (w *wrappedLogger) Debug(args ...any) { logArgs(w.slog, slog.LevelDebug, args...) }
func (w *wrappedLogger) Info(args ...any)  { logArgs(w.slog, slog.LevelInfo, args...) }
func (w *wrappedLogger) Warn(args ...any)  { logArgs(w.slog, slog.LevelWarn, args...) }
func (w *wrappedLogger) Error(args ...any) { logArgs(w.slog, slog.LevelError, args...) }

func (w *wrappedLogger) WithFields(fields ...any) iface.MessageLogger {
	return &wrappedLogger{slog: w.slog.With(getFields(fields...)...)}
}

func (w *wrappedLogger) Nested(fields ...any) iface.Logger {
	return &wrappedLogger{slog: w.slog.With(getFields(fields...)...)}
}

func logf(l *slog.Logger, level slog.Level, format string, args ...any) {
	l.Log(context.Background(), level, fmt.Sprintf(format, args...))
}

func logArgs(l *slog.Logger, level slog.Level, args ...any) {
	l.Log(context.Background(), level, fmt.Sprint(args...))
}

// getFields flattens variadic field arguments into a key/value []any suitable
// for slog.Logger.With. Standalone iface.Fields maps may appear anywhere in
// the parameters and are merged in. Mirrors adapter/logrus.getFields.
func getFields(fields ...any) []any {
	merged := make(iface.Fields)
	out := make([]any, 0, len(fields))
	offset := 0
	for i, val := range fields {
		if fieldsMap, ok := val.(iface.Fields); ok {
			maps.Copy(merged, fieldsMap)
			offset++
			continue
		}
		if (i-offset)%2 != 0 {
			out = append(out, fmt.Sprintf("%s", fields[i-1]), val)
		}
	}
	for k, v := range merged {
		out = append(out, k, v)
	}
	return out
}
