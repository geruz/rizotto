package metrics

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/geruz/rizotto/logger"
)

func InitMetrics(ctx context.Context, httpAddr string) {
	startServer(ctx, httpAddr)
}

func startServer(ctx context.Context, httpAddr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	const readHeaderTimeout = 10 * time.Second

	srv := &http.Server{ //nolint:exhaustruct
		Addr:              httpAddr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		logger.Info(ctx, "Starting metrics server on "+httpAddr)
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(ctx, "metrics server failed", err)
		}
	}()
}
