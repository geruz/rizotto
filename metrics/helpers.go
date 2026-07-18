package metrics

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var meter = otel.Meter("github.com/geruz/rizotto/metrics")

var DurationBuckets_5ms_10s = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

func Counter(name, help string) metric.Int64Counter {
	counter, err := meter.Int64Counter(name, metric.WithDescription(help))
	if err != nil {
		panic(err)
	}

	return counter
}

func DurationHistogram(name, help string, bucketBoundaries []float64) metric.Float64Histogram {
	histogram, err := meter.Float64Histogram(
		name,
		metric.WithDescription(help),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(bucketBoundaries...),
	)
	if err != nil {
		panic(err)
	}

	return histogram
}
