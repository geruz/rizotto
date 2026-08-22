package gateway

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/geruz/rizotto/logger"
)

type route[TContext ExtendedHTTPContext, TRequest any, TResponse any] struct {
	method string
	path   string

	createContext    func(HTTPContext) (TContext, HTTPError)
	controllerMethod func(TContext, TRequest) (TResponse, HTTPError)
	renderer         Renderer[TContext, TResponse]
}

func (r route[TContext, TRequest, TResult]) Bind(s HTTPGateway) {
	logger.Info(context.Background(), "Binding route "+r.path)
	s.router.HandleFunc(r.method+" "+r.path, r.Handler())
}

func (route route[TContext, TRequest, TResult]) Handler() http.HandlerFunc {
	requestBuilder := requestObjBuilder[TRequest]()

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		httpCtx := HTTPContext{
			Context: ctx,
			Writer:  w,
			Request: r,
		}
		tCtx, guardErr := route.createContext(httpCtx)
		if guardErr != nil {
			// The typed context does not exist yet, so the renderer cannot be
			// used and the error goes straight to the writer.
			err := RenderHTTPError(w, guardErr)
			if err != nil {
				logger.Error(ctx, "failed to render error response", err)
			}

			return
		}

		reqObj, err := requestBuilder(w, r)

		if err != nil {
			err := route.renderer.RenderError(tCtx, err)
			if err != nil {
				logger.Error(ctx, "failed to render error response: %v", err)
			}

			return
		}

		result, err := route.controllerMethod(tCtx, reqObj)
		if err != nil {
			err := route.renderer.RenderError(tCtx, err)
			if err != nil {
				logger.Error(ctx, "failed to render error response: %v", err)
			}

			return
		}

		renderErr := route.renderer.RenderSuccess(tCtx, result)
		if renderErr != nil {
			logger.Error(ctx, "failed to render success response: %v", renderErr)
		}
	}
}

func (r route[TContext, TRequest, TResult]) Method() string {
	return r.method
}

func (r route[TContext, TRequest, TResult]) Path() string {
	return r.path
}

func (r route[TContext, TRequest, TResult]) String() string {
	return r.method + " " + r.path
}

func (r route[TContext, TRequest, TResult]) ConfigureControllerMethod(
	m func(TContext, TRequest) (TResult, HTTPError),
) Route {
	r.controllerMethod = m

	return r
}

type Route interface {
	Bind(s HTTPGateway)
	Handler() http.HandlerFunc
	Method() string
	Path() string
	String() string
}

// MustMakeRouteMethod builds one route. The area function may reject the request:
// when it answers an error the handler is never called and the error is rendered
// as is, which is how an authenticated area answers 401 or 403.
func MustMakeRouteMethod[TRequest any, TContext ExtendedHTTPContext, TResponse any](
	routeStr string,
	context func(ctx HTTPContext) (TContext, HTTPError),
	handler func(ctx TContext, req TRequest) (TResponse, HTTPError),
	renderer Renderer[TContext, TResponse],
) Route {
	l := strings.Split(routeStr, " ")
	const expectedPartsCount = 2
	if len(l) != expectedPartsCount {
		panic("invalid route string")
	}

	return route[TContext, TRequest, TResponse]{
		method: l[0],
		path:   l[1],

		createContext:    context,
		controllerMethod: handler,
		renderer:         renderer,
	}
}

type RouteTable []Route

func (r RouteTable) RouteStrings() []string {
	routeStrings := make([]string, len(r))
	for index, route := range r {
		routeStrings[index] = route.String()
	}

	return routeStrings
}

var ErrRouteNotFound = errors.New("route not found")

func (r RouteTable) Find(routeStr string) (Route, error) {
	for _, route := range r {
		if route.String() == routeStr {
			return route, nil
		}
	}

	return nil, ErrRouteNotFound
}

func JoinRouteTables(tables ...RouteTable) RouteTable {
	var r RouteTable
	for _, table := range tables {
		r = append(r, table...)
	}

	return r
}
