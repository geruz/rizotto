package api

import (
	"errors"
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
	Body        struct{}
	QueryParams string
)

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

func (h RequestConfiguration) WithQuery(params string) RequestConfiguration {
	h.configurations = append(h.configurations, QueryParams(params))

	return h
}
