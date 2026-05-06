package charm

import (
	"bytes"
	"io"
	"strings"
	"testing"

	log "github.com/charmbracelet/log"
	"github.com/stretchr/testify/require"

	iface "github.com/anchore/go-logger"
)

func newTestLogger(buf io.Writer, level iface.Level) iface.Logger {
	return New(Config{
		Level:     level,
		Output:    buf,
		Formatter: log.TextFormatter,
	})
}

func TestNew_levelFiltering(t *testing.T) {
	tests := []struct {
		name      string
		level     iface.Level
		emit      func(iface.Logger)
		wantToken string
		wantEmpty bool
	}{
		{
			name:      "info passes at info level",
			level:     iface.InfoLevel,
			emit:      func(l iface.Logger) { l.Info("hello") },
			wantToken: "hello",
		},
		{
			name:      "debug filtered at info level",
			level:     iface.InfoLevel,
			emit:      func(l iface.Logger) { l.Debug("hello") },
			wantEmpty: true,
		},
		{
			name:      "trace passes at trace level",
			level:     iface.TraceLevel,
			emit:      func(l iface.Logger) { l.Trace("traced") },
			wantToken: "traced",
		},
		{
			name:      "trace filtered at debug level",
			level:     iface.DebugLevel,
			emit:      func(l iface.Logger) { l.Trace("traced") },
			wantEmpty: true,
		},
		{
			name:      "disabled drops all",
			level:     iface.DisabledLevel,
			emit:      func(l iface.Logger) { l.Error("nope") },
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			l := newTestLogger(buf, tt.level)
			tt.emit(l)

			got := buf.String()
			if tt.wantEmpty {
				require.Empty(t, got)
				return
			}
			require.Contains(t, got, tt.wantToken)
		})
	}
}

func TestNew_messageConstruction(t *testing.T) {
	tests := []struct {
		name string
		emit func(iface.Logger)
		want string
	}{
		{
			name: "info joins string args without space",
			emit: func(l iface.Logger) { l.Info("a", 1, "b") },
			want: "a1b",
		},
		{
			name: "info inserts space between non-string operands",
			emit: func(l iface.Logger) { l.Info(1, 2, 3) },
			want: "1 2 3",
		},
		{
			name: "infof formats template",
			emit: func(l iface.Logger) { l.Infof("hello %s=%d", "k", 7) },
			want: "hello k=7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			l := newTestLogger(buf, iface.InfoLevel)
			tt.emit(l)
			require.Contains(t, buf.String(), tt.want)
		})
	}
}

func TestNew_withFieldsAndNested(t *testing.T) {
	buf := &bytes.Buffer{}
	l := newTestLogger(buf, iface.InfoLevel)

	t.Run("WithFields with key/value pairs", func(t *testing.T) {
		buf.Reset()
		l.WithFields("k1", "v1", "k2", 2).Info("msg")
		got := buf.String()
		require.Contains(t, got, "msg")
		require.Contains(t, got, "k1=v1")
		require.Contains(t, got, "k2=2")
	})

	t.Run("WithFields with iface.Fields map", func(t *testing.T) {
		buf.Reset()
		l.WithFields(iface.Fields{"a": "x"}).Info("m")
		require.Contains(t, buf.String(), "a=x")
	})

	t.Run("Nested propagates fields across calls", func(t *testing.T) {
		buf.Reset()
		nested := l.Nested("scope", "child")
		nested.Info("m1")
		nested.Warn("m2")
		got := buf.String()
		require.Equal(t, 2, strings.Count(got, "scope=child"))
	})

	t.Run("mixed fields and key/value pairs", func(t *testing.T) {
		buf.Reset()
		l.WithFields("k", "v", iface.Fields{"a": "b"}, "c", "d").Info("m")
		got := buf.String()
		require.Contains(t, got, "k=v")
		require.Contains(t, got, "a=b")
		require.Contains(t, got, "c=d")
	})
}

func TestController(t *testing.T) {
	first := &bytes.Buffer{}
	second := &bytes.Buffer{}

	l := newTestLogger(first, iface.InfoLevel)
	ctl, ok := l.(iface.Controller)
	require.True(t, ok)

	require.Same(t, first, ctl.GetOutput())

	l.Info("first-line")
	require.Contains(t, first.String(), "first-line")

	ctl.SetOutput(second)
	require.Same(t, second, ctl.GetOutput())

	l.Info("second-line")
	require.Contains(t, second.String(), "second-line")
	require.NotContains(t, first.String(), "second-line")
}

func TestUse_implementsController(t *testing.T) {
	buf := &bytes.Buffer{}
	base := log.NewWithOptions(buf, log.Options{Level: LevelTrace, Formatter: log.TextFormatter})

	l := Use(base)
	l.Info("hello")
	require.Contains(t, buf.String(), "hello")

	ctl, ok := l.(iface.Controller)
	require.True(t, ok, "Use-returned logger must implement iface.Controller")

	other := &bytes.Buffer{}
	ctl.SetOutput(other)
	l.Info("redirected")
	require.Contains(t, other.String(), "redirected")
}

func TestUse_traceRequiresLoggerLevel(t *testing.T) {
	tests := []struct {
		name      string
		level     log.Level
		wantTrace bool
	}{
		{"logger at LevelTrace emits trace", LevelTrace, true},
		{"logger at LevelDebug suppresses trace", log.DebugLevel, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			base := log.NewWithOptions(buf, log.Options{Level: tt.level, Formatter: log.TextFormatter})
			l := Use(base)
			l.Trace("traced-msg")
			if tt.wantTrace {
				require.Contains(t, buf.String(), "traced-msg")
				return
			}
			require.Empty(t, buf.String())
		})
	}
}

func TestTranslateLevel(t *testing.T) {
	tests := []struct {
		in   iface.Level
		want log.Level
	}{
		{iface.TraceLevel, LevelTrace},
		{iface.DebugLevel, log.DebugLevel},
		{iface.InfoLevel, log.InfoLevel},
		{iface.WarnLevel, log.WarnLevel},
		{iface.ErrorLevel, log.ErrorLevel},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, translateLevel(tt.in))
	}

	require.True(t, translateLevel(iface.DisabledLevel) > log.ErrorLevel)
}

func TestDefaultConfig_outputDefaultsToStderr(t *testing.T) {
	l := New(Config{Level: iface.InfoLevel})
	ctl, ok := l.(iface.Controller)
	require.True(t, ok)
	require.NotNil(t, ctl.GetOutput())
}
