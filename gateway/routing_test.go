package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// userContext is the typed context of a route that needs an authenticated caller.
type userContext struct {
	HTTPContext

	User string
}

func (c userContext) GetHTTPContext() HTTPContext {
	return c.HTTPContext
}

type emptyRequest struct{}

type answer struct {
	User string `json:"user"`
}

func openArea(ctx HTTPContext) (userContext, HTTPError) {
	return userContext{HTTPContext: ctx, User: "john"}, nil
}

func call(t *testing.T, route Route) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/me", nil)
	route.Handler()(recorder, request)

	return recorder
}

func TestRouteCallsTheHandlerWhenTheAreaAccepts(t *testing.T) {
	t.Parallel()

	route := JSONMethod("GET /api/v1/me", openArea,
		func(ctx userContext, _ emptyRequest) (answer, HTTPError) {
			return answer{User: ctx.User}, nil
		})

	recorder := call(t, route)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var body answer
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&body))
	assert.Equal(t, "john", body.User)
}

// TestRouteRejectsBeforeTheHandler pins the point of the fallible area function:
// an unauthenticated caller is answered without the handler ever running.
func TestRouteRejectsBeforeTheHandler(t *testing.T) {
	t.Parallel()

	handlerRan := false
	closedArea := func(_ HTTPContext) (userContext, HTTPError) {
		return userContext{HTTPContext: HTTPContext{}, User: ""}, //nolint:exhaustruct_v5 // the rejected context is unused
			NewUnauthorizedError("no session")
	}

	route := JSONMethod("GET /api/v1/me", closedArea,
		func(_ userContext, _ emptyRequest) (answer, HTTPError) {
			handlerRan = true

			return answer{User: "leaked"}, nil
		})

	recorder := call(t, route)

	assert.False(t, handlerRan, "the handler must not run for a rejected request")
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var body HTTPApiError
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&body))
	assert.Equal(t, UnauthorizedErrorCode, body.ErrorCode)
	assert.Equal(t, "no session", body.Message)
}

func TestRedirectRenderSendsTheCallerAway(t *testing.T) {
	t.Parallel()

	route := Redirect("GET /api/v1/me", openArea,
		func(_ userContext, _ emptyRequest) (Redirection, HTTPError) {
			return Redirection{Location: "https://accounts.example.com/authorize", Status: 0}, nil
		})

	recorder := call(t, route)

	assert.Equal(t, http.StatusFound, recorder.Code, "a zero Status must default to 302")
	assert.Equal(t, "https://accounts.example.com/authorize", recorder.Header().Get("Location"))
}

func TestRedirectRenderKeepsAnExplicitStatus(t *testing.T) {
	t.Parallel()

	route := Redirect("GET /api/v1/me", openArea,
		func(_ userContext, _ emptyRequest) (Redirection, HTTPError) {
			return Redirection{Location: "/app", Status: http.StatusSeeOther}, nil
		})

	recorder := call(t, route)

	assert.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Equal(t, "/app", recorder.Header().Get("Location"))
}

func TestMustMakeRouteMethodPanicsOnAMalformedRoute(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		JSONMethod("/api/v1/me", openArea,
			func(_ userContext, _ emptyRequest) (answer, HTTPError) {
				return answer{User: ""}, nil
			})
	})
}
