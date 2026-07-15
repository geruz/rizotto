package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var Percentiles_60_90_99 = map[float64]float64{
	0.5:  0.05,  //nolint:mnd
	0.9:  0.01,  //nolint:mnd
	0.99: 0.001, //nolint:mnd
}

func SummaryVec(
	name string,
	help string,
	objectives map[float64]float64,
	labels []string,
) *prometheus.SummaryVec {
	return prometheus.NewSummaryVec(
		prometheus.SummaryOpts{ //nolint:exhaustruct
			Name:       name,
			Help:       help,
			Objectives: objectives,
		},
		labels,
	)
}
