package errors

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWith_CopiesAndKeepsOriginal(t *testing.T) {
	original := New(TypeInvalidInput, CodeInvalidInput, "invalid filter").WithAdditional("first")

	changed := original.
		WithURL("https://signoz.io/docs/filters").
		WithAdditional("second", "third").
		WithSuggestiveAdditional("unknown key servce", "did you mean: service").
		WithSuggestions("narrow the time range").
		WithRetryAfter(5 * time.Second)

	assert.Equal(t, []Additional{{Message: "first"}}, original.Additionals())
	assert.Empty(t, original.URL())
	assert.Empty(t, original.Suggestions())
	assert.Zero(t, original.RetryAfter())

	assert.Equal(t, "https://signoz.io/docs/filters", changed.URL())
	assert.Equal(t, []Additional{
		{Message: "first"},
		{Message: "second"},
		{Message: "third"},
		{Message: "unknown key servce", Suggestions: []string{"did you mean: service"}},
	}, changed.Additionals())
	assert.Equal(t, []string{"narrow the time range"}, changed.Suggestions())
	assert.Equal(t, 5*time.Second, changed.RetryAfter())
	assert.Equal(t, original.Stacktrace(), changed.Stacktrace())
}

func TestWith_SharedParent_NoAliasing(t *testing.T) {
	parent := New(TypeInvalidInput, CodeInvalidInput, "invalid query").WithAdditional("a", "b")

	left := parent.WithAdditional("left")
	right := parent.WithAdditional("right")

	assert.Equal(t, "left", left.Additionals()[2].Message)
	assert.Equal(t, "right", right.Additionals()[2].Message)
}

func TestWithStacktrace_Replaces(t *testing.T) {
	err := New(TypeFatal, CodeFatal, "nil map write").WithStacktrace("goroutine 7 [running]:\nmain.handler()\n")

	assert.Equal(t, "goroutine 7 [running]:\nmain.handler()\n", err.Stacktrace())
}
