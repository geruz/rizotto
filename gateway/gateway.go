package gateway

import (
	"context"
	"net/http"
	"time"

	"github.com/geruz/rizotto/logger"
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

	srv := &http.Server{ //nolint:exhaustruct_v5
		Addr:              addr,
		Handler:           s.router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	return srv.ListenAndServe()
}

func (s HTTPGateway) Get(path string, handler http.HandlerFunc) HTTPGateway {
	return s
}

func (s HTTPGateway) Patch(path string, handler http.HandlerFunc) HTTPGateway {
	return s
}

func (s HTTPGateway) Static(path string, dir string) HTTPGateway {
	return s
}

func (s HTTPGateway) Routing(methods []Route) HTTPGateway {
	for _, method := range methods {
		method.Bind(s)
	}

	return s
}
