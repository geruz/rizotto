package gateway

import (
	"errors"
	"net/http"
	"testing"

	"github.com/geruz/rizotto/bb"
	"github.com/stretchr/testify/assert"
)

// errPlain is an error belonging to no group, so no matcher may claim it.
var errPlain = errors.New("something else")

// TestIfUnauthorizedMatchesTheServiceError pins that the matcher identifies the
// error the bb layer already answers, so a service saying "unauthorized" becomes
// a 401 instead of a 500.
func TestIfUnauthorizedMatchesTheServiceError(t *testing.T) {
	t.Parallel()

	unauthorized := NewUnauthorizedError("no session")
	mapped := MapError(bb.NewUnauthorizedError("token expired")).
		IfNotFound(NewNotFoundError("user_not_found", "User not found", "")).
		IfUnauthorized(unauthorized).
		Others(NewInternalError())

	assert.Equal(t, http.StatusUnauthorized, mapped.StatusCode())
}

func TestIfUnauthorizedIgnoresOtherErrors(t *testing.T) {
	t.Parallel()

	mapped := MapError(bb.NewNotFoundError("user not found")).
		IfUnauthorized(NewUnauthorizedError("no session")).
		Others(NewInternalError())

	assert.Equal(t, http.StatusInternalServerError, mapped.StatusCode())

	mapped = MapError(errPlain).
		IfUnauthorized(NewUnauthorizedError("no session")).
		Others(NewInternalError())

	assert.Equal(t, http.StatusInternalServerError, mapped.StatusCode())
}

func TestIfNotFoundStillMatches(t *testing.T) {
	t.Parallel()

	notFound := NewNotFoundError("user_not_found", "User not found", "")
	mapped := MapError(bb.NewNotFoundError("user not found")).
		IfNotFound(notFound).
		IfUnauthorized(NewUnauthorizedError("no session")).
		Others(NewInternalError())

	assert.Equal(t, http.StatusNotFound, mapped.StatusCode())
}

func TestNewForbiddenError(t *testing.T) {
	t.Parallel()

	forbidden := NewForbiddenError("not_an_admin", "Forbidden", "admins only")

	assert.Equal(t, http.StatusForbidden, forbidden.StatusCode())
}
