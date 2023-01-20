package logrus

import (
	"github.com/sirupsen/logrus"

	iface "github.com/anchore/go-logger"
)

var _ iface.Logger = (*nestedLogger)(nil)

// nestedLogger is a wrapper for Logrus to enable nested logging configuration (loggers that always attach key-value pairs to all log entries)
type nestedLogger struct {
	entry *logrus.Entry
}

// Tracef takes a formatted template string and template arguments for the trace logging level.
func (l *nestedLogger) Tracef(format string, args ...interface{}) {
	l.entry.Tracef(format, args...)
}

// Debugf takes a formatted template string and template arguments for the debug logging level.
func (l *nestedLogger) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}

// Infof takes a formatted template string and template arguments for the info logging level.
func (l *nestedLogger) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}

// Warnf takes a formatted template string and template arguments for the warning logging level.
func (l *nestedLogger) Warnf(format string, args ...interface{}) {
	l.entry.Warnf(format, args...)
}

// Errorf takes a formatted template string and template arguments for the error logging level.
func (l *nestedLogger) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
}

// Trace logs the given arguments at the trace logging level.
func (l *nestedLogger) Trace(args ...interface{}) {
	l.entry.Trace(args...)
}

// Debug logs the given arguments at the debug logging level.
func (l *nestedLogger) Debug(args ...interface{}) {
	l.entry.Debug(args...)
}

// Info logs the given arguments at the info logging level.
func (l *nestedLogger) Info(args ...interface{}) {
	l.entry.Info(args...)
}

// Warn logs the given arguments at the warning logging level.
func (l *nestedLogger) Warn(args ...interface{}) {
	l.entry.Warn(args...)
}

// Error logs the given arguments at the error logging level.
func (l *nestedLogger) Error(args ...interface{}) {
	l.entry.Error(args...)
}

// WithFields returns a message entry with multiple key-value fields.
func (l *nestedLogger) WithFields(fields ...interface{}) iface.MessageLogger {
	return l.entry.WithFields(getFields(fields...))
}

func (l *nestedLogger) Nested(fields ...interface{}) iface.Logger {
	return &nestedLogger{entry: l.entry.WithFields(getFields(fields...))}
}
