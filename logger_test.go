package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevelFromVerbosity(t *testing.T) {
	tests := []struct {
		name   string
		v      int
		levels []Level
		want   Level
	}{
		{
			name:   "no configured levels disables logging",
			v:      0,
			levels: []Level{},
			want:   DisabledLevel,
		},
		{
			name:   "no configured levels disables logging (with negative verbosity)",
			v:      -1,
			levels: []Level{},
			want:   DisabledLevel,
		},

		{
			name: "negative verbosity selects the lowest level",
			v:    -10,
			levels: []Level{
				WarnLevel, InfoLevel, DebugLevel, TraceLevel,
			},
			want: WarnLevel,
		},
		{
			name: "select lowest level",
			v:    0,
			levels: []Level{
				WarnLevel, InfoLevel, DebugLevel, TraceLevel,
			},
			want: WarnLevel,
		},
		{
			name: "positive valid verbosity selects correct level index",
			v:    1,
			levels: []Level{
				WarnLevel, InfoLevel, DebugLevel, TraceLevel,
			},
			want: InfoLevel,
		},
		{
			name: "select highest level index",
			v:    3,
			levels: []Level{
				WarnLevel, InfoLevel, DebugLevel, TraceLevel,
			},
			want: TraceLevel,
		},
		{
			name: "select edge of bounds",
			v:    4,
			levels: []Level{
				WarnLevel, InfoLevel, DebugLevel, TraceLevel,
			},
			want: TraceLevel,
		},
		{
			name: "select out of bounds",
			v:    5,
			levels: []Level{
				WarnLevel, InfoLevel, DebugLevel, TraceLevel,
			},
			want: TraceLevel,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LevelFromVerbosity(tt.v, tt.levels...)
			assert.Equal(t, tt.want, got)
		})
	}
}
