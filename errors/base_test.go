package errors

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWith_CopiesAndKeepsOriginal(t *testing.T) {
	original := New(TypeInvalidInput, MustNewCode("invalid_input"), "invalid filter").WithAdditional("first")

	changed := original.
		WithURL("https://signoz.io/docs/filters").
		WithAdditional("second").WithAdditional("third").
		WithAdditional("unknown key servce", "did you mean: service").
		WithSuggestions("narrow the time range").
		WithRetryAfter(5 * time.Second)

	assert.Equal(t, []additional{{message: "first"}}, original.additionals)
	assert.Empty(t, original.URL())
	assert.Empty(t, original.Suggestions())
	assert.Zero(t, original.RetryAfter())

	assert.Equal(t, "https://signoz.io/docs/filters", changed.URL())
	assert.Equal(t, []additional{
		{message: "first"},
		{message: "second"},
		{message: "third"},
		{message: "unknown key servce", suggestions: []string{"did you mean: service"}},
	}, changed.additionals)
	assert.Equal(t, []string{"narrow the time range"}, changed.Suggestions())
	assert.Equal(t, 5*time.Second, changed.RetryAfter())
	assert.Equal(t, original.Stacktrace(), changed.Stacktrace())
}

func TestWith_SharedParent_NoAliasing(t *testing.T) {
	parent := New(TypeInvalidInput, MustNewCode("invalid_input"), "invalid query").WithAdditional("a").WithAdditional("b")

	left := parent.WithAdditional("left")
	right := parent.WithAdditional("right")

	leftAdditionals := left.additionals
	rightAdditionals := right.additionals
	require.Len(t, leftAdditionals, 3)
	require.Len(t, rightAdditionals, 3)

	assert.Equal(t, "left", leftAdditionals[2].message)
	assert.Equal(t, "right", rightAdditionals[2].message)
}

func TestWithStacktrace_Replaces(t *testing.T) {
	err := New(TypeFatal, MustNewCode("fatal"), "nil map write").WithStacktrace("goroutine 7 [running]:\nmain.handler()\n")

	assert.Equal(t, "goroutine 7 [running]:\nmain.handler()\n", err.Stacktrace())
}

func TestWithRetryAfter_NegativeSetsNone(t *testing.T) {
	err := New(TypeTooManyRequests, MustNewCode("rate_limited"), "slow down").WithRetryAfter(-time.Second)

	assert.Zero(t, err.RetryAfter())
	assert.Zero(t, RetryAfterOf(err))
	assert.Nil(t, AsException(err).Retry)
}
