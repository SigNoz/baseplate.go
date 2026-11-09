package errors

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsException(t *testing.T) {
	testCases := []struct {
		name         string
		err          error
		expectedJSON string
	}{
		{
			name:         "Base_EmptyListsNotNull",
			err:          New(TypeNotFound, MustNewCode("not_found"), "dashboard not found"),
			expectedJSON: `{"type":"not-found","code":"not_found","message":"dashboard not found","errors":[],"suggestions":[]}`,
		},
		{
			name: "Base_AllDetails",
			err: New(TypeTooManyRequests, MustNewCode("rate_limited"), "slow down").
				WithURL("https://signoz.io/docs/limits").
				WithAdditional("100 requests per minute").
				WithAdditional("key servce", "did you mean: service").
				WithSuggestions("batch the requests").
				WithRetryAfter(2 * time.Second),
			expectedJSON: `{"type":"too-many-requests","code":"rate_limited","message":"slow down","url":"https://signoz.io/docs/limits",` +
				`"errors":[{"message":"100 requests per minute","suggestions":[]},{"message":"key servce","suggestions":["did you mean: service"]}],` +
				`"retry":{"delay":2000000000},"suggestions":["batch the requests"]}`,
		},
		{
			name:         "Wrapped_OwnMessageOnly",
			err:          Wrap(context.Canceled, TypeCanceled, MustNewCode("canceled"), "query canceled"),
			expectedJSON: `{"type":"canceled","code":"canceled","message":"query canceled","errors":[],"suggestions":[]}`,
		},
		{
			name:         "Foreign_UnknownWithFullText",
			err:          context.DeadlineExceeded,
			expectedJSON: `{"type":"unknown","code":"unknown","message":"context deadline exceeded","errors":[],"suggestions":[]}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			body, err := json.Marshal(AsException(testCase.err))
			require.NoError(t, err)

			assert.JSONEq(t, testCase.expectedJSON, string(body))
		})
	}
}

func TestAsException_DoesNotShareSlices(t *testing.T) {
	err := New(TypeInvalidInput, MustNewCode("invalid_input"), "invalid query").
		WithAdditional("key servce", "did you mean: service").
		WithSuggestions("check the syntax")

	body := AsException(err)
	require.Len(t, body.Errors, 1)
	require.Len(t, body.Errors[0].Suggestions, 1)
	require.Len(t, body.Suggestions, 1)

	body.Errors[0].Suggestions[0] = "changed"
	body.Suggestions[0] = "changed"

	assert.Equal(t, []Additional{{Message: "key servce", Suggestions: []string{"did you mean: service"}}}, err.Additionals())
	assert.Equal(t, []string{"check the syntax"}, err.Suggestions())
}
