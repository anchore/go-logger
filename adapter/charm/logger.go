// Package charm provides a forward adapter that exposes an iface.Logger
// backed by a charmbracelet/log *log.Logger.
package charm

import (
	"fmt"
	"io"
	"maps"
	"math"
	"os"

	log "github.com/charmbracelet/log"

	iface "github.com/anchore/go-logger"
)

// LevelTrace is the charm log level used by this adapter to represent
// iface.TraceLevel. It is one step below log.DebugLevel. Callers building
// their own *log.Logger for use with Use must configure their logger's
// minimum level at or below LevelTrace if they want trace logging emitted.
const LevelTrace log.Level = log.DebugLevel - 4

// Config contains all configurable values for building a charm-backed logger.
type Config struct {
	Level           iface.Level
	Output          io.Writer
	Formatter       log.Formatter
	TimeFormat      string
	ReportCaller    bool
	ReportTimestamp bool
	Prefix          string
}

func DefaultConfig() Config {
	return Config{
		Level:     iface.InfoLevel,
		Output:    os.Stderr,
		Formatter: log.TextFormatter,
	}
}

var (
	_ iface.Logger     = (*logger)(nil)
	_ iface.Controller = (*logger)(nil)
)

// logger adapts a *log.Logger to iface.Logger. The same struct backs both
// New- and Use-constructed loggers because charm exposes SetOutput, which is
// all we need to satisfy iface.Controller. We track the writer ourselves
// because charm/log does not expose a getter for it.
type logger struct {
	log    *log.Logger
	output io.Writer
}

// New builds a *log.Logger from cfg and adapts it.
func New(cfg Config) iface.Logger {
	out := cfg.Output
	if out == nil {
		out = os.Stderr
	}
	l := log.NewWithOptions(out, log.Options{
		Level:           translateLevel(cfg.Level),
		Formatter:       cfg.Formatter,
		TimeFormat:      cfg.TimeFormat,
		ReportCaller:    cfg.ReportCaller,
		ReportTimestamp: cfg.ReportTimestamp,
		Prefix:          cfg.Prefix,
	})
	return &logger{log: l, output: out}
}

// Use wraps an already-configured *log.Logger. GetOutput will return nil
// until the caller (or this adapter via SetOutput) supplies a writer, since
// charm/log does not expose a way to read the current writer back.
func Use(l *log.Logger) iface.Logger {
	return &logger{log: l}
}

func translateLevel(l iface.Level) log.Level {
	switch l {
	case iface.TraceLevel:
		return LevelTrace
	case iface.DebugLevel:
		return log.DebugLevel
	case iface.InfoLevel:
		return log.InfoLevel
	case iface.WarnLevel:
		return log.WarnLevel
	case iface.ErrorLevel:
		return log.ErrorLevel
	case iface.DisabledLevel:
		return log.Level(math.MaxInt32)
	}
	return log.InfoLevel
}

func (l *logger) Tracef(format string, args ...any) {
	l.log.Log(LevelTrace, fmt.Sprintf(format, args...))
}
func (l *logger) Debugf(format string, args ...any) { l.log.Debug(fmt.Sprintf(format, args...)) }
func (l *logger) Infof(format string, args ...any)  { l.log.Info(fmt.Sprintf(format, args...)) }
func (l *logger) Warnf(format string, args ...any)  { l.log.Warn(fmt.Sprintf(format, args...)) }
func (l *logger) Errorf(format string, args ...any) { l.log.Error(fmt.Sprintf(format, args...)) }

func (l *logger) Trace(args ...any) { l.log.Log(LevelTrace, fmt.Sprint(args...)) }
func (l *logger) Debug(args ...any) { l.log.Debug(fmt.Sprint(args...)) }
func (l *logger) Info(args ...any)  { l.log.Info(fmt.Sprint(args...)) }
func (l *logger) Warn(args ...any)  { l.log.Warn(fmt.Sprint(args...)) }
func (l *logger) Error(args ...any) { l.log.Error(fmt.Sprint(args...)) }

func (l *logger) WithFields(fields ...any) iface.MessageLogger {
	return &logger{log: l.log.With(getFields(fields...)...), output: l.output}
}

func (l *logger) Nested(fields ...any) iface.Logger {
	return &logger{log: l.log.With(getFields(fields...)...), output: l.output}
}

func (l *logger) SetOutput(w io.Writer) {
	l.output = w
	l.log.SetOutput(w)
}

func (l *logger) GetOutput() io.Writer { return l.output }

// getFields flattens variadic field arguments into a key/value []any suitable
// for log.Logger.With. Standalone iface.Fields maps may appear anywhere in
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
