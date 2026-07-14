package gateway

import (
	"context"
	"net/http"
)

type HTTPContext struct {
	context.Context //nolint: containedctx

	Writer  http.ResponseWriter
	Request *http.Request
}

type ExtendedHTTPContext interface {
	GetHTTPContext() HTTPContext
}

func (c HTTPContext) GetHTTPContext() HTTPContext {
	return c
}
