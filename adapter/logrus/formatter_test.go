package logrus

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type terminalBuffer struct {
	bytes.Buffer
}

func (t *terminalBuffer) IsTerminal() bool { return true }

func TestTextFormatter_ColorsFollowOutput(t *testing.T) {
	l := logrus.New()
	l.SetFormatter(DefaultTextFormatter())

	// a plain buffer is not a terminal, so no colors
	plain := &bytes.Buffer{}
	l.SetOutput(plain)
	l.Warn("first")
	assert.NotContains(t, plain.String(), "\x1b[")

	// swapping to a terminal-aware writer after the first entry should enable colors
	term := &terminalBuffer{}
	l.SetOutput(term)
	l.Warn("second")
	assert.Contains(t, term.String(), "\x1b[")
}

func Test_extractPrefix(t *testing.T) {

	tests := []struct {
		name   string
		msg    string
		prefix string
		rest   string
	}{
		{
			name:   "no prefix",
			msg:    "hello world",
			prefix: "",
			rest:   "hello world",
		},
		{
			name:   "prefix",
			msg:    "[0000] hello world",
			prefix: "0000",
			rest:   "hello world",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, rest := extractPrefix(tt.msg)
			assert.Equal(t, tt.prefix, prefix)
			assert.Equal(t, tt.rest, rest)
		})
	}
}
