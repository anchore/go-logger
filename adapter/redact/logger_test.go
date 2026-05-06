package redact

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anchore/go-logger"
	slogadapter "github.com/anchore/go-logger/adapter/slog"
)

func Test_RedactingLogger(t *testing.T) {
	tests := []struct {
		name   string
		redact []string
	}{
		{
			name:   "single value",
			redact: []string{"joe"},
		},
		{
			name:   "multi value",
			redact: []string{"bob", "alice"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buff := bytes.Buffer{}
			out := slogadapter.New(slogadapter.Config{
				Level:  logger.TraceLevel,
				Output: &buff,
				Format: slogadapter.FormatText,
			})
			require.Implements(t, (*logger.Controller)(nil), out)

			redactor := New(out, NewStore(test.redact...))

			var fieldObj = make(logger.Fields)
			for _, v := range test.redact {
				fieldObj[v] = v
			}

			var format strings.Builder
			var fields []any
			for _, v := range test.redact {
				fields = append(fields, v)
				format.WriteString("%s")
			}

			fields = append(fields, 3)
			format.WriteString("%d")

			fields = append(fields, int32(3))
			format.WriteString("%d")

			fields = append(fields, 3.2)
			format.WriteString("%f")

			fields = append(fields, float32(4.3))
			format.WriteString("%f")

			fields = append(fields, fieldObj)
			format.WriteString("%+v")

			var interlacedFields []any
			for i, f := range fields {
				interlacedFields = append(interlacedFields, fmt.Sprintf("%d", i), f)
			}

			nestedFieldLogger := redactor.Nested(interlacedFields...).WithFields(interlacedFields...)

			nestedFieldLogger.Tracef(format.String(), fields...)
			nestedFieldLogger.Trace(fields...)

			nestedFieldLogger.Debugf(format.String(), fields...)
			nestedFieldLogger.Debug(fields...)

			nestedFieldLogger.Infof(format.String(), fields...)
			nestedFieldLogger.Info(fields...)

			nestedFieldLogger.Warnf(format.String(), fields...)
			nestedFieldLogger.Warn(fields...)

			nestedFieldLogger.Errorf(format.String(), fields...)
			nestedFieldLogger.Error(fields...)

			result := buff.String()

			// this is a string indicator that we've coerced an instance to a new type that does not match the format type (e.g. %d)
			assert.NotContains(t, result, "%")

			assert.NotEmpty(t, result)
			for _, v := range test.redact {
				assert.NotContains(t, result, v)
			}
		})
	}
}
