package gateway

import (
	"encoding/json"
	"io"
	"net/http"
)

type Renderer[TContext ExtendedHTTPContext, TResponse any] interface {
	RenderSuccess(ctx TContext, response TResponse) error
	RenderError(ctx TContext, response HTTPError) error
}

type JSONRender[HTTPContext ExtendedHTTPContext, TResponse any] struct{}

func (jsonRender JSONRender[HTTPContext, TResponse]) RenderSuccess(ctx HTTPContext, r TResponse) error {
	w := ctx.GetHTTPContext().Writer
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return json.NewEncoder(w).Encode(r)
}

func (jsonRender JSONRender[HTTPContext, TResponse]) RenderError(ctx HTTPContext, httpError HTTPError) error {
	w := ctx.GetHTTPContext().Writer
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpError.StatusCode())

	return json.NewEncoder(w).Encode(httpError.ErrorObj())
}

type ContentRender[HTTPContext ExtendedHTTPContext, TResponse ContentProvider] struct{}

func (fr ContentRender[HTTPContext, TResponse]) RenderSuccess(ctx HTTPContext, file TResponse) error {
	w := ctx.GetHTTPContext().Writer
	w.Header().Set("Content-Type", file.ContentType())
	// w.Header().Set("Content-Length", strconv.Itoa(file.ContentLength))
	w.WriteHeader(http.StatusOK)

	return file.Write(w)
	//	_, err := io.Copy(w, file.Content)
	// return err
	// return json.NewEncoder(w).Encode(r)
}

func (fr ContentRender[HTTPContext, TResponse]) RenderError(ctx HTTPContext, httpError HTTPError) error {
	w := ctx.GetHTTPContext().Writer
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(httpError.StatusCode())

	return json.NewEncoder(w).Encode(httpError.ErrorObj())
}

func JSONMethod[TContext ExtendedHTTPContext, TRequest any, TResponse any](
	routeStr string,
	context func(ctx HTTPContext) TContext,
	handler func(ctx TContext, req TRequest) (TResponse, HTTPError),
) Route {
	renderer := JSONRender[TContext, TResponse]{}

	return MustMakeRouteMethod(routeStr, context, handler, renderer)
}

type ContentProvider interface {
	ContentType() string
	Write(bw io.Writer) error
}

func Content[TRequest any, TResponse ContentProvider, TContext ExtendedHTTPContext](
	routeStr string,
	context func(ctx HTTPContext) TContext,
	handler func(ctx TContext, req TRequest) (TResponse, HTTPError),
) Route {
	renderer := ContentRender[TContext, TResponse]{}

	return MustMakeRouteMethod(routeStr, context, handler, renderer)
}
