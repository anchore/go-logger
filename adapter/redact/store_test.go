package redact

import (
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRedactor is a simple redactor for testing
type mockRedactor struct {
	redactFunc func(string) string
	idValue    string
}

func (m *mockRedactor) RedactString(s string) string {
	return m.redactFunc(s)
}

func (m *mockRedactor) id() string {
	return m.idValue
}

func TestNewStore(t *testing.T) {
	tests := []struct {
		name           string
		initialValues  []string
		testInput      string
		expectedOutput string
	}{
		{
			name:           "empty store",
			initialValues:  nil,
			testInput:      "no redaction here",
			expectedOutput: "no redaction here",
		},
		{
			name:           "store with initial values",
			initialValues:  []string{"secret", "password"},
			testInput:      "my secret and password",
			expectedOutput: "my ******* and *******",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(tt.initialValues...)

			require.NotNil(t, store)

			actual := store.RedactString(tt.testInput)
			assert.Equal(t, tt.expectedOutput, actual)
		})
	}
}

func TestStore_Add(t *testing.T) {
	tests := []struct {
		name           string
		initialValues  []string
		addValues      []string
		testInput      string
		expectedOutput string
	}{
		{
			name:           "add single value",
			initialValues:  nil,
			addValues:      []string{"secret"},
			testInput:      "this is secret",
			expectedOutput: "this is *******",
		},
		{
			name:           "add multiple values",
			initialValues:  nil,
			addValues:      []string{"secret", "password", "token"},
			testInput:      "secret password token",
			expectedOutput: "******* ******* *******",
		},
		{
			name:           "add to existing store",
			initialValues:  []string{"existing"},
			addValues:      []string{"new"},
			testInput:      "existing and new",
			expectedOutput: "******* and *******",
		},
		{
			name:           "ignore single character",
			initialValues:  nil,
			addValues:      []string{"a"},
			testInput:      "a message",
			expectedOutput: "a message",
		},
		{
			name:           "ignore empty string",
			initialValues:  nil,
			addValues:      []string{""},
			testInput:      "no redaction",
			expectedOutput: "no redaction",
		},
		{
			name:           "add duplicate values",
			initialValues:  []string{"secret"},
			addValues:      []string{"secret", "secret"},
			testInput:      "secret message",
			expectedOutput: "******* message",
		},
		{
			name:           "two character value is valid",
			initialValues:  nil,
			addValues:      []string{"ab"},
			testInput:      "ab test",
			expectedOutput: "******* test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(tt.initialValues...)
			store.Add(tt.addValues...)

			actual := store.RedactString(tt.testInput)
			assert.Equal(t, tt.expectedOutput, actual)
		})
	}
}

func TestStore_RedactString(t *testing.T) {
	tests := []struct {
		name           string
		redactions     []string
		input          string
		expectedOutput string
	}{
		{
			name:           "redact single occurrence",
			redactions:     []string{"secret"},
			input:          "this is a secret",
			expectedOutput: "this is a *******",
		},
		{
			name:           "redact multiple occurrences",
			redactions:     []string{"secret"},
			input:          "secret secret secret",
			expectedOutput: "******* ******* *******",
		},
		{
			name:           "redact multiple different values",
			redactions:     []string{"secret", "password"},
			input:          "my secret is not the password",
			expectedOutput: "my ******* is not the *******",
		},
		{
			name:           "no redaction needed",
			redactions:     []string{"secret"},
			input:          "no sensitive data",
			expectedOutput: "no sensitive data",
		},
		{
			name:           "empty string",
			redactions:     []string{"secret"},
			input:          "",
			expectedOutput: "",
		},
		{
			name:           "redact substring",
			redactions:     []string{"pass"},
			input:          "password",
			expectedOutput: "*******word",
		},
		{
			name:           "case sensitive redaction",
			redactions:     []string{"Secret"},
			input:          "secret and Secret",
			expectedOutput: "secret and *******",
		},
		{
			name:           "redact special characters",
			redactions:     []string{"p@ssw0rd!"},
			input:          "my p@ssw0rd! is strong",
			expectedOutput: "my ******* is strong",
		},
		{
			// overlapping redactions: order matters, shorter match may prevent longer match
			name:       "overlapping redaction values",
			redactions: []string{"secret", "secretkey"},
			input:      "my secretkey and secret",
			// note: if "secret" is replaced first, "secretkey" becomes "*******key"
			// the actual output depends on iteration order of the set
			expectedOutput: "my *******key and *******",
		},
		{
			name:           "redaction with whitespace",
			redactions:     []string{"my secret"},
			input:          "this is my secret phrase",
			expectedOutput: "this is ******* phrase",
		},
		{
			name:           "unicode redaction values",
			redactions:     []string{"🔑password", "秘密"},
			input:          "my 🔑password and 秘密 data",
			expectedOutput: "my ******* and ******* data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(tt.redactions...)

			actual := store.RedactString(tt.input)
			assert.Equal(t, tt.expectedOutput, actual)
		})
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store := NewStore("initial")

	var wg sync.WaitGroup
	numGoroutines := 100

	// test concurrent Add operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			store.Add("secret" + strconv.Itoa(idx))
		}(i)
	}

	// test concurrent RedactString operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = store.RedactString("some text with initial value")
		}()
	}

	wg.Wait()

	// verify the store still works correctly after concurrent access
	result := store.RedactString("initial")
	assert.Equal(t, "*******", result)
}

func TestNewRedactorCollection(t *testing.T) {
	tests := []struct {
		name           string
		redactors      []Redactor
		testInput      string
		expectedOutput string
	}{
		{
			name: "single redactor",
			redactors: []Redactor{
				&mockRedactor{
					redactFunc: func(s string) string {
						return strings.ReplaceAll(s, "secret", "***")
					},
					idValue: "mock1",
				},
			},
			testInput:      "my secret",
			expectedOutput: "my ***",
		},
		{
			name: "multiple redactors applied in sequence",
			redactors: []Redactor{
				&mockRedactor{
					redactFunc: func(s string) string {
						return strings.ReplaceAll(s, "secret", "***")
					},
					idValue: "mock1",
				},
				&mockRedactor{
					redactFunc: func(s string) string {
						return strings.ReplaceAll(s, "password", "###")
					},
					idValue: "mock2",
				},
			},
			testInput:      "my secret password",
			expectedOutput: "my *** ###",
		},
		{
			name: "deduplicate redactors by id",
			redactors: []Redactor{
				&mockRedactor{
					redactFunc: func(s string) string {
						return strings.ReplaceAll(s, "secret", "***")
					},
					idValue: "same-id",
				},
				&mockRedactor{
					redactFunc: func(s string) string {
						return strings.ReplaceAll(s, "password", "###")
					},
					idValue: "same-id",
				},
			},
			testInput:      "my secret password",
			expectedOutput: "my *** password",
		},
		{
			name: "flatten nested collections",
			redactors: []Redactor{
				newRedactorCollection(
					&mockRedactor{
						redactFunc: func(s string) string {
							return strings.ReplaceAll(s, "secret", "***")
						},
						idValue: "mock1",
					},
					&mockRedactor{
						redactFunc: func(s string) string {
							return strings.ReplaceAll(s, "password", "###")
						},
						idValue: "mock2",
					},
				),
				&mockRedactor{
					redactFunc: func(s string) string {
						return strings.ReplaceAll(s, "token", "@@@")
					},
					idValue: "mock3",
				},
			},
			testInput:      "secret password token",
			expectedOutput: "*** ### @@@",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collection := newRedactorCollection(tt.redactors...)

			actual := collection.RedactString(tt.testInput)
			assert.Equal(t, tt.expectedOutput, actual)
		})
	}
}

func TestRedactorCollection_EmptyCollection(t *testing.T) {
	collection := newRedactorCollection()

	input := "test string"
	output := collection.RedactString(input)

	// empty collection should not modify the string
	assert.Equal(t, input, output)
}

func TestStore_InRedactorCollection(t *testing.T) {
	// test that Store works correctly when used as a Redactor in a collection
	store := NewStore("password")
	mockRedactor := &mockRedactor{
		redactFunc: func(s string) string {
			return strings.ReplaceAll(s, "secret", "###")
		},
		idValue: "mock1",
	}

	collection := newRedactorCollection(store, mockRedactor)

	input := "my secret password"
	expected := "my ### *******"
	actual := collection.RedactString(input)

	assert.Equal(t, expected, actual)
}

func TestStore_SequentialAdds(t *testing.T) {
	store := NewStore()

	// add values one at a time
	store.Add("first")
	result1 := store.RedactString("first second third")
	assert.Equal(t, "******* second third", result1)

	store.Add("second")
	result2 := store.RedactString("first second third")
	assert.Equal(t, "******* ******* third", result2)

	store.Add("third")
	result3 := store.RedactString("first second third")
	assert.Equal(t, "******* ******* *******", result3)
}

func TestNewStore_WithDuplicates(t *testing.T) {
	// test that duplicates in constructor are handled correctly
	store := NewStore("secret", "password", "secret", "password")

	input := "my secret and password"
	expected := "my ******* and *******"
	actual := store.RedactString(input)

	assert.Equal(t, expected, actual)
}
