package logger

import (
	"io"
)

type Level string

const (
	ErrorLevel Level = "error"
	WarnLevel  Level = "warn"
	InfoLevel  Level = "info"
	DebugLevel Level = "debug"
	TraceLevel Level = "trace"
)

type Logger interface {
	MessageLogger
	FieldLogger
	NestedLogger
}

type Controller interface {
	SetOutput(io.Writer)
	GetOutput() io.Writer
}

type NestedLogger interface {
	Nested(fields ...interface{}) Logger
}

type FieldLogger interface {
	WithFields(fields ...interface{}) MessageLogger
}

type MessageLogger interface {
	ErrorLogger
	WarnLogger
	InfoLogger
	DebugLogger
	TraceLogger
}

type ErrorLogger interface {
	Errorf(format string, args ...interface{})
	Error(args ...interface{})
}

type WarnLogger interface {
	Warnf(format string, args ...interface{})
	Warn(args ...interface{})
}

type InfoLogger interface {
	Infof(format string, args ...interface{})
	Info(args ...interface{})
}

type DebugLogger interface {
	Debugf(format string, args ...interface{})
	Debug(args ...interface{})
}

type TraceLogger interface {
	Tracef(format string, args ...interface{})
	Trace(args ...interface{})
}
