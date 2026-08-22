package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// applyAll runs the configurators the way TestAPICall does, so the test pins what
// actually reaches the area function of a route.
func applyAll(t *testing.T, configure func(RequestConfiguration) RequestConfiguration) *http.Request {
	t.Helper()

	//nolint:exhaustruct_v5 // only the configurators matter here
	cfg := configure(RequestConfiguration{})

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/me", nil)

	for _, configuration := range cfg.configurations {
		require.NoError(t, configuration.apply(request))
	}

	return request
}

func TestWithHeader(t *testing.T) {
	t.Parallel()

	request := applyAll(t, func(cfg RequestConfiguration) RequestConfiguration {
		return cfg.WithHeader("X-Request-Id", "abc")
	})

	assert.Equal(t, "abc", request.Header.Get("X-Request-Id"))
}

func TestWithBearer(t *testing.T) {
	t.Parallel()

	request := applyAll(t, func(cfg RequestConfiguration) RequestConfiguration {
		return cfg.WithBearer("session-token")
	})

	assert.Equal(t, "Bearer session-token", request.Header.Get("Authorization"))
}

func TestWithCookie(t *testing.T) {
	t.Parallel()

	request := applyAll(t, func(cfg RequestConfiguration) RequestConfiguration {
		return cfg.WithCookie("session", "session-token")
	})

	cookie, err := request.Cookie("session")
	require.NoError(t, err)
	assert.Equal(t, "session-token", cookie.Value)
}
