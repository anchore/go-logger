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
	ErrorMessageLogger
	WarnMessageLogger
	InfoMessageLogger
	DebugMessageLogger
	TraceMessageLogger
}

//type MessageLogger interface {
//	Logf(level Level, format string, args ...interface{})
//	Log(level Level, args ...interface{})
//}

type ErrorMessageLogger interface {
	Errorf(format string, args ...interface{})
	Error(args ...interface{})
}

type WarnMessageLogger interface {
	Warnf(format string, args ...interface{})
	Warn(args ...interface{})
}

type InfoMessageLogger interface {
	Infof(format string, args ...interface{})
	Info(args ...interface{})
}

type DebugMessageLogger interface {
	Debugf(format string, args ...interface{})
	Debug(args ...interface{})
}

type TraceMessageLogger interface {
	Tracef(format string, args ...interface{})
	Trace(args ...interface{})
}
