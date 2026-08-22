package bb

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_patchContextByCustomHeaders_ShouldPatchHeaders(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		headerName      string
		headerValue     string
		shouldBePatched bool
	}{
		{
			name:            "x- headers",
			headerName:      "X-test-header",
			headerValue:     "X test value",
			shouldBePatched: true,
		},

		{
			name:            "other headers",
			headerName:      "Other-test-header",
			headerValue:     "Other test value",
			shouldBePatched: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange.
			req := http.Request{Header: http.Header{}} //nolint:exhaustruct_v5
			req.Header.Set(tc.headerName, tc.headerValue)

			// Act.
			patchedCtx := patchContextByCustomHeaders(context.Background(), &req)

			// Assert.
			rawHeaderValues := patchedCtx.Value(CustomHeaders("customHeaders"))
			headerValues, ok := rawHeaderValues.(map[string]string)
			assert.True(t, ok, "custom headers should be a map")

			if tc.shouldBePatched {
				assert.Len(t, headerValues, 1, "custom headers should contain the patched header")
				assert.Contains(t, headerValues, strings.ToLower(tc.headerName),
					"custom headers should contain %v=%v", tc.headerName, tc.headerValue)
			} else {
				assert.Empty(t, headerValues, "%v should not have been patched", tc.headerName)
			}
		})
	}
}
