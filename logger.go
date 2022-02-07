package logger

import "io"

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
	Errorf(format string, args ...interface{})
	Error(args ...interface{})
	Warnf(format string, args ...interface{})
	Warn(args ...interface{})
	Infof(format string, args ...interface{})
	Info(args ...interface{})
	Debugf(format string, args ...interface{})
	Debug(args ...interface{})
}
