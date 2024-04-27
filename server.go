package rizotto

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/geruz/rizotto/logger"
	"github.com/geruz/rizotto/settings/env"
)

type HTTPRouter interface {
	http.Handler
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

type HTTPGateway struct {
	router HTTPRouter
}

func NewHTTPGateway() HTTPGateway {
	return HTTPGateway{
		router: http.NewServeMux(),
	}
}

func (s HTTPGateway) ListenAndServe(ctx context.Context, addr string) error {
	logger.Info(ctx, "Starting server on "+addr)

	const readHeaderTimeout = 10 * time.Second

	srv := &http.Server{ //nolint:exhaustruct
		Addr:              addr,
		Handler:           s.router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	return srv.ListenAndServe()
}

func (s HTTPGateway) Get(path string, handler http.HandlerFunc) HTTPGateway {
	return s
}

type route[TContext ExtendedHTTPContext, TRequest any, TResponse any] struct {
	method string
	path   string

	createContext    func(HTTPContext) TContext
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
		tCtx := route.createContext(httpCtx)

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

type ExtendedHTTPContext interface {
	GetHTTPContext() HTTPContext
}

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

func (s HTTPGateway) Patch(path string, handler http.HandlerFunc) HTTPGateway {
	return s
}

func (s HTTPGateway) Static(path string, dir string) HTTPGateway {
	return s
}

type Route interface {
	Bind(s HTTPGateway)
	Handler() http.HandlerFunc
	Method() string
	Path() string
	String() string
}

func MustMakeRouteMethod[TRequest any, TContext ExtendedHTTPContext, TResponse any](
	routeStr string,
	context func(ctx HTTPContext) TContext,
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
		//		pipe:   []interface{}{context, handler, renderer},
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

func (s HTTPGateway) Routing(methods []Route) HTTPGateway {
	for _, method := range methods {
		method.Bind(s)
	}

	return s
}

func MustInitEnv(ctx context.Context) {
	env.MustLoadEnvFile(".env")
}

func MustInitLogger(ctx context.Context) {
	logger.MustInit()
}

func InitMetrics(ctx context.Context, addr string) {
}

func InitScheduler(ctx context.Context) {
}

func InitTracer(ctx context.Context) {
}

type Controller struct {
	routeTable RouteTable
}

func NewController(routeTable RouteTable) Controller {
	return Controller{
		routeTable: routeTable,
	}
}

func (c *Controller) AddRoutes(routes ...Route) Controller {
	c.routeTable = append(c.routeTable, routes...)

	return *c
}

func (r Controller) RouteTable() RouteTable {
	return r.routeTable
}

type (
	HTTPError interface {
		ErrorObj() any
		StatusCode() int
	}
	APIError interface {
		Error() string
	}
	HTTPContext struct {
		context.Context //nolint: containedctx

		Writer  http.ResponseWriter
		Request *http.Request
	}

	HTTPApiError struct {
		Code      int       `json:"code"`
		Message   string    `json:"message"`
		ErrorCode ErrorCode `json:"errorCode"`
		Details   string    `json:"details"`
	}
	ErrorCode string
)

const (
	InternalErrorCode       ErrorCode = "internal_error"
	InvalidRequestErrorCode ErrorCode = "invalid_request"
)

func NoAuth(ctx HTTPContext) HTTPContext {
	return ctx
}

func (e HTTPApiError) StatusCode() int {
	return e.Code
}

func (e HTTPApiError) ErrorObj() any {
	return e
}

func NewInternalError() HTTPError {
	return HTTPApiError{
		Code:      http.StatusInternalServerError,
		Message:   "Internal error",
		ErrorCode: InternalErrorCode,
		Details:   "Internal error",
	}
}

func NewInvalidRequest(message string) HTTPError {
	return HTTPApiError{
		Code:      http.StatusBadRequest,
		Message:   message,
		ErrorCode: InvalidRequestErrorCode,
		Details:   "Invalid request",
	}
}

func NewNotFoundError(errorCode ErrorCode, message string, details string) HTTPError {
	return HTTPApiError{
		Code:      http.StatusNotFound,
		Message:   message,
		ErrorCode: errorCode,
		Details:   details,
	}
}

type ConfigurableHTTPError struct {
	OriginalError error
	matched       HTTPError
}

func (c ConfigurableHTTPError) Error() string {
	return c.OriginalError.Error()
}

type ErrorGroup interface {
	IsNotFound() bool
}

func (c ConfigurableHTTPError) IfNotFound(err HTTPError) ConfigurableHTTPError {
	if err == nil {
		return c
	}

	if e, ok := c.OriginalError.(ErrorGroup); ok && e.IsNotFound() {
		c.matched = err
	}

	return c
}

func (c ConfigurableHTTPError) Others(err HTTPError) HTTPError {
	if c.matched != nil {
		return c.matched
	}

	return err
}

func MapError(err error) ConfigurableHTTPError {
	return ConfigurableHTTPError{
		OriginalError: err,
		matched:       nil,
	}
}

func (c HTTPContext) GetHTTPContext() HTTPContext {
	return c
}

type (
	PermissionDenied struct{}
	NotFound         struct{}
)
