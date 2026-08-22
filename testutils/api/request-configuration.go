package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/geruz/rizotto/documentation/openapi"
	"github.com/geruz/rizotto/err"
)

type (
	PathParams struct {
		key   string
		value string
	}
	Body struct {
		contentType string
		payload     any
	}
	QueryParams string
	Header      struct {
		key   string
		value string
	}
	CookieParam struct {
		name  string
		value string
	}
)

const jsonContentType = "application/json"

var ErrMissingPathVariable = errors.New("path variable not found in route")

func (p PathParams) apply(req *http.Request) error {
	if !strings.Contains(req.URL.Path, "{"+p.key+"}") {
		return err.Wrap("key = '"+p.key+"'", ErrMissingPathVariable)
	}

	req.URL.Path = strings.ReplaceAll(req.URL.Path, "{"+p.key+"}", p.value)
	req.SetPathValue(p.key, p.value)

	return nil
}

func (b Body) apply(req *http.Request) error {
	if b.payload == nil {
		return nil
	}

	raw, err := json.Marshal(b.payload)
	if err != nil {
		return err
	}

	req.Body = io.NopCloser(bytes.NewReader(raw))
	req.ContentLength = int64(len(raw))
	req.Header.Set("Content-Type", b.contentType)

	return nil
}

func (h Header) apply(req *http.Request) error {
	req.Header.Set(h.key, h.value)

	return nil
}

func (c CookieParam) apply(req *http.Request) error {
	// A cookie travelling on a request carries only its name and value: Secure,
	// HttpOnly and SameSite are instructions to the browser, set by the answer.
	//nolint:exhaustruct_v5,gosec // G124 is about response cookies
	req.AddCookie(&http.Cookie{
		Name:  c.name,
		Value: c.value,
	})

	return nil
}

func (q QueryParams) apply(req *http.Request) error {
	query, err := url.ParseQuery(string(q))
	if err != nil {
		return err
	}

	qq := req.URL.Query()

	for key, values := range query {
		for _, value := range values {
			qq.Add(key, value)
		}
	}

	qStr := qq.Encode()
	req.URL.RawQuery = qStr

	return nil
}

type RequestConfiguration struct {
	routeStr       string
	configurations []configurator

	doc *openapi.Operation
}

type configurator interface {
	apply(h *http.Request) error
}

func (h RequestConfiguration) WithPathVar(key string, value string) RequestConfiguration {
	h.configurations = append(h.configurations, PathParams{
		key:   key,
		value: value,
	})

	return h
}

// WithJSONBody sends the payload as the json body of the request.
func (h RequestConfiguration) WithJSONBody(payload any) RequestConfiguration {
	h.configurations = append(h.configurations, Body{
		contentType: jsonContentType,
		payload:     payload,
	})

	return h
}

func (h RequestConfiguration) WithQuery(params string) RequestConfiguration {
	h.configurations = append(h.configurations, QueryParams(params))

	return h
}

// WithHeader sends one header with the request.
func (h RequestConfiguration) WithHeader(key, value string) RequestConfiguration {
	h.configurations = append(h.configurations, Header{
		key:   key,
		value: value,
	})

	return h
}

// WithBearer authenticates the request with a bearer token.
func (h RequestConfiguration) WithBearer(token string) RequestConfiguration {
	return h.WithHeader("Authorization", "Bearer "+token)
}

// WithCookie sends one cookie with the request, which is how a session reaches an
// area function in a route test.
func (h RequestConfiguration) WithCookie(name, value string) RequestConfiguration {
	h.configurations = append(h.configurations, CookieParam{
		name:  name,
		value: value,
	})

	return h
}
