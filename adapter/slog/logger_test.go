package slog

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	iface "github.com/anchore/go-logger"
)

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
			l := New(Config{Level: tt.level, Output: buf, Format: FormatText})
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
			name: "info joins args with fmt.Sprint",
			emit: func(l iface.Logger) { l.Info("a", 1, "b") },
			// fmt.Sprint adds a space only when neither operand is a string;
			// here every adjacency includes a string, so no spaces are inserted.
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
			l := New(Config{Level: iface.InfoLevel, Output: buf, Format: FormatText})
			tt.emit(l)
			require.Contains(t, buf.String(), tt.want)
		})
	}
}

func TestNew_withFieldsAndNested(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(Config{Level: iface.InfoLevel, Output: buf, Format: FormatText})

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
		got := buf.String()
		require.Contains(t, got, "a=x")
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

	l := New(Config{Level: iface.InfoLevel, Output: first, Format: FormatText})
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

func TestUse_doesNotImplementController(t *testing.T) {
	buf := &bytes.Buffer{}
	base := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: LevelTrace}))

	l := Use(base)
	l.Info("hello")
	require.Contains(t, buf.String(), "hello")

	_, ok := l.(iface.Controller)
	require.False(t, ok, "Use-returned logger must not implement iface.Controller")
}

func TestUse_traceRequiresHandlerLevel(t *testing.T) {
	tests := []struct {
		name      string
		handler   func(io.Writer) slog.Handler
		wantTrace bool
	}{
		{
			name:      "handler at LevelTrace emits trace",
			handler:   func(w io.Writer) slog.Handler { return slog.NewTextHandler(w, &slog.HandlerOptions{Level: LevelTrace}) },
			wantTrace: true,
		},
		{
			name:      "handler at LevelDebug suppresses trace",
			handler:   func(w io.Writer) slog.Handler { return slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}) },
			wantTrace: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			l := Use(slog.New(tt.handler(buf)))
			l.Trace("traced-msg")
			if tt.wantTrace {
				require.Contains(t, buf.String(), "traced-msg")
				return
			}
			require.Empty(t, buf.String())
		})
	}
}

func TestJSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(Config{Level: iface.InfoLevel, Output: buf, Format: FormatJSON})
	l.WithFields("k", "v").Info("hello")

	got := buf.String()
	require.Contains(t, got, `"msg":"hello"`)
	require.Contains(t, got, `"k":"v"`)
}

func TestDefaultConfig_outputDefaultsToStderr(t *testing.T) {
	cfg := Config{Level: iface.InfoLevel}
	l := New(cfg)
	ctl, ok := l.(iface.Controller)
	require.True(t, ok)
	// New normalizes nil Output to os.Stderr.
	require.NotNil(t, ctl.GetOutput())
}

func TestTranslateLevel(t *testing.T) {
	tests := []struct {
		in   iface.Level
		want slog.Level
	}{
		{iface.TraceLevel, LevelTrace},
		{iface.DebugLevel, slog.LevelDebug},
		{iface.InfoLevel, slog.LevelInfo},
		{iface.WarnLevel, slog.LevelWarn},
		{iface.ErrorLevel, slog.LevelError},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, translateLevel(tt.in))
	}

	// DisabledLevel maps to a level that even Error is below.
	require.True(t, translateLevel(iface.DisabledLevel) > slog.LevelError)
}

// ensure context is wired correctly (no panic on nil context inside slog.Log)
func TestLogfDoesNotPanic(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(Config{Level: iface.InfoLevel, Output: buf, Format: FormatText})
	require.NotPanics(t, func() { l.Infof("x=%d", 1) })
	_ = context.Background()
}
