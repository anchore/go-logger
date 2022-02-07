package logrus

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"

	iface "github.com/anchore/go-logger"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var _ iface.Logger = (*logger)(nil)
var _ iface.Controller = (*logger)(nil)

const defaultLogFilePermissions fs.FileMode = 0644

// Config contains all configurable values for the Logrus entry
type Config struct {
	EnableConsole bool
	EnableFile    bool
	Structured    bool
	Level         logrus.Level
	FileLocation  string
}

// logger contains all runtime values for using Logrus with the configured output target and input configuration values.
type logger struct {
	config Config
	logger *logrus.Logger
	output io.Writer
}

// New creates a new entry with the given configuration
func New(cfg Config) (iface.Logger, error) {
	l := logrus.New()

	var output io.Writer
	switch {
	case cfg.EnableConsole && cfg.EnableFile:
		logFile, err := os.OpenFile(cfg.FileLocation, os.O_WRONLY|os.O_CREATE, defaultLogFilePermissions)
		if err != nil {
			return nil, fmt.Errorf("unable to setup log file: %w", err)
		}
		output = io.MultiWriter(os.Stderr, logFile)
	case cfg.EnableConsole:
		output = os.Stderr
	case cfg.EnableFile:
		logFile, err := os.OpenFile(cfg.FileLocation, os.O_WRONLY|os.O_CREATE, defaultLogFilePermissions)
		if err != nil {
			return nil, fmt.Errorf("unable to setup log file: %w", err)
		}
		output = logFile
	default:
		output = ioutil.Discard
	}

	l.SetOutput(output)
	l.SetLevel(cfg.Level)

	if cfg.Structured {
		l.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat:   "2006-01-02 15:04:05",
			DisableTimestamp:  false,
			DisableHTMLEscape: false,
			PrettyPrint:       false,
		})
	} else {
		l.SetFormatter(&prefixed.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
			ForceFormatting: true,
		})
	}

	return &logger{
		config: cfg,
		logger: l,
		output: output,
	}, nil
}

// Debugf takes a formatted template string and template arguments for the debug logging level.
func (l *logger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

// Infof takes a formatted template string and template arguments for the info logging level.
func (l *logger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

// Warnf takes a formatted template string and template arguments for the warning logging level.
func (l *logger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

// Errorf takes a formatted template string and template arguments for the error logging level.
func (l *logger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

// Debug logs the given arguments at the debug logging level.
func (l *logger) Debug(args ...interface{}) {
	l.logger.Debug(args...)
}

// Info logs the given arguments at the info logging level.
func (l *logger) Info(args ...interface{}) {
	l.logger.Info(args...)
}

// Warn logs the given arguments at the warning logging level.
func (l *logger) Warn(args ...interface{}) {
	l.logger.Warn(args...)
}

// Error logs the given arguments at the error logging level.
func (l *logger) Error(args ...interface{}) {
	l.logger.Error(args...)
}

// WithFields returns a message entry with multiple key-value fields.
func (l *logger) WithFields(fields ...interface{}) iface.MessageLogger {
	return l.logger.WithFields(getFields(fields...))
}

func (l *logger) Nested(fields ...interface{}) iface.Logger {
	return &nestedLogger{entry: l.logger.WithFields(getFields(fields...))}

}

func (l *logger) SetOutput(writer io.Writer) {
	l.output = writer
	l.logger.SetOutput(writer)
}

func (l *logger) GetOutput() io.Writer {
	return l.output
}

func getFields(fields ...interface{}) logrus.Fields {
	f := make(logrus.Fields)
	for i, val := range fields {
		if i%2 != 0 {
			f[fmt.Sprintf("%s", fields[i-1])] = val
		}
	}
	return f
}
