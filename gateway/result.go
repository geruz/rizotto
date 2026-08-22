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

// RenderHTTPError writes an error response directly to the writer. The renderers
// below need the typed context of the route, which does not exist yet when the
// area function of a guarded route rejects the request.
func RenderHTTPError(w http.ResponseWriter, httpError HTTPError) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpError.StatusCode())

	return json.NewEncoder(w).Encode(httpError.ErrorObj())
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

// JSONMethod declares a route answering JSON. The area function authenticates the
// caller and may answer 401 or 403 before the handler runs.
func JSONMethod[TContext ExtendedHTTPContext, TRequest any, TResponse any](
	routeStr string,
	context func(ctx HTTPContext) (TContext, HTTPError),
	handler func(ctx TContext, req TRequest) (TResponse, HTTPError),
) Route {
	renderer := JSONRender[TContext, TResponse]{}

	return MustMakeRouteMethod(routeStr, context, handler, renderer)
}

type ContentProvider interface {
	ContentType() string
	Write(bw io.Writer) error
}

// Redirection is the answer of a handler that sends the caller elsewhere, the two
// legs of an OAuth flow among them. A zero Status redirects with 302.
type Redirection struct {
	Location string
	Status   int
}

type RedirectRender[HTTPContext ExtendedHTTPContext] struct{}

func (rr RedirectRender[HTTPContext]) RenderSuccess(ctx HTTPContext, redirection Redirection) error {
	w := ctx.GetHTTPContext().Writer
	w.Header().Set("Location", redirection.Location)

	status := redirection.Status
	if status == 0 {
		status = http.StatusFound
	}

	w.WriteHeader(status)

	return nil
}

func (rr RedirectRender[HTTPContext]) RenderError(ctx HTTPContext, httpError HTTPError) error {
	return RenderHTTPError(ctx.GetHTTPContext().Writer, httpError)
}

// Redirect declares a route whose handler answers a Redirection.
func Redirect[TRequest any, TContext ExtendedHTTPContext](
	routeStr string,
	context func(ctx HTTPContext) (TContext, HTTPError),
	handler func(ctx TContext, req TRequest) (Redirection, HTTPError),
) Route {
	renderer := RedirectRender[TContext]{}

	return MustMakeRouteMethod(routeStr, context, handler, renderer)
}

func Content[TRequest any, TResponse ContentProvider, TContext ExtendedHTTPContext](
	routeStr string,
	context func(ctx HTTPContext) (TContext, HTTPError),
	handler func(ctx TContext, req TRequest) (TResponse, HTTPError),
) Route {
	renderer := ContentRender[TContext, TResponse]{}

	return MustMakeRouteMethod(routeStr, context, handler, renderer)
}
