package trace

import (
	"bufio"
	"errors"
	"net"
	"net/http"

	"github.com/geruz/rizotto/crypto/token"
	"github.com/geruz/rizotto/logger"
)

type loggingResponseWriter struct {
	http.ResponseWriter

	statusCode int
}

var ErrNotHijacker = errors.New("writer is not a hijacker")

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, 0}
}

func (l *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if v, ok := l.ResponseWriter.(http.Hijacker); ok {
		return v.Hijack()
	}

	return nil, nil, ErrNotHijacker
}

func (l *loggingResponseWriter) StatusCode() int {
	return l.statusCode
}

func (l *loggingResponseWriter) WriteHeader(code int) {
	l.statusCode = code
	l.ResponseWriter.WriteHeader(code)
}

const requestIDLength = 32

func TraceMiddleware(spanName string) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = token.Generate(r.Context(), requestIDLength)
			}
			ctx := logger.WithParams(r.Context(), map[string]string{
				"requestID": requestID,
			})
			ctx, span := Span(ctx, spanName)
			w.Header().Add("X-Request-ID", requestID)
			span.SetAttributes(
				String("http.method", r.Method),
				String("http.url", r.URL.String()),
				String("http.requestID", requestID),
			)
			defer span.End()
			lrw := NewLoggingResponseWriter(w)
			h.ServeHTTP(lrw, r.WithContext(ctx))
			span.SetAttributes(
				Int("http.status.code", lrw.statusCode),
			)
		})
	}
}
