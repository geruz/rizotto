package gateway

import (
	"net/http"
	"time"
)

func SetCookie(ctx ExtendedHTTPContext, cookie *http.Cookie) {
	http.SetCookie(ctx.GetHTTPContext().Writer, cookie)
}

func Cookie(ctx ExtendedHTTPContext, name string) (string, bool) {
	cookie, err := ctx.GetHTTPContext().Request.Cookie(name)
	if err != nil {
		return "", false
	}

	return cookie.Value, true
}

func NewSessionCookie(name, value string, ttl time.Duration) *http.Cookie {
	return &http.Cookie{ //nolint:exhaustruct_v5
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

func NewExpiredCookie(name string) *http.Cookie {
	return &http.Cookie{ //nolint:exhaustruct_v5
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}
