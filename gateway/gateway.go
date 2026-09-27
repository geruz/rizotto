package gateway

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	pathpkg "path"
	"strings"
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

// Static serves the files of dir under path, as a single page application build
// needs them: a page the browser asks for and dir does not have gets index.html,
// so that the client-side router can take it, while a missing asset or a request
// of an API client still gets 404. Routes registered for more specific paths,
// such as /api/..., take precedence over it.
func (s HTTPGateway) Static(path string, dir string) HTTPGateway {
	prefix := strings.TrimSuffix(path, "/")

	logger.Info(context.Background(), "Serving "+dir+" on "+prefix+"/")
	s.router.HandleFunc("GET "+prefix+"/", staticHandler(prefix, os.DirFS(dir)))

	return s
}

func staticHandler(prefix string, root fs.FS) http.HandlerFunc {
	files := http.StripPrefix(prefix, http.FileServerFS(root))

	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(pathpkg.Clean("/"+strings.TrimPrefix(r.URL.Path, prefix)), "/")
		if name == "" {
			files.ServeHTTP(w, r)

			return
		}

		info, err := fs.Stat(root, name)
		if err == nil && !info.IsDir() {
			files.ServeHTTP(w, r)

			return
		}

		if pathpkg.Ext(name) == "" && strings.Contains(r.Header.Get("Accept"), "text/html") {
			http.ServeFileFS(w, r, root, "index.html")

			return
		}

		http.NotFound(w, r)
	}
}

func (s HTTPGateway) Routing(methods []Route) HTTPGateway {
	for _, method := range methods {
		method.Bind(s)
	}

	return s
}
