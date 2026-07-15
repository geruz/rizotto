package sql

import (
	"github.com/geruz/rizotto/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var sqlQueryDuration = metrics.SummaryVec(
	"sql_query_durations_seconds",
	"SQL query durations in seconds.",
	metrics.Percentiles_60_90_99,
	[]string{"name"},
)

func init() {
	prometheus.MustRegister(sqlQueryDuration)
}
