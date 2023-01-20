package logrus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
