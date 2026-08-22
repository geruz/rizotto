package metrics

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/geruz/rizotto/logger"
)

func MustInitMetrics(ctx context.Context, httpAddr string) {
	mustInitMeterProvider(ctx)
	startServer(ctx, httpAddr)
}

// mustInitMeterProvider wires the OpenTelemetry MeterProvider to a Prometheus
// exporter, which registers itself as a collector on the default Prometheus
// registry so promhttp.Handler can serve it on /metrics.
func mustInitMeterProvider(ctx context.Context) {
	exporter, err := otelprometheus.New()
	if err != nil {
		logger.Error(ctx, "failed to initialize prometheus exporter", err)

		return
	}

	otel.SetMeterProvider(sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	))
}

func startServer(ctx context.Context, httpAddr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	const readHeaderTimeout = 10 * time.Second

	srv := &http.Server{ //nolint:exhaustruct_v5
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
