package pg

import (
	"github.com/geruz/rizotto/metrics"
)

var sqlQueryDuration = metrics.DurationHistogram(
	"sql_query_durations_seconds",
	"SQL query durations in seconds.",
	metrics.DurationBuckets_5ms_10s,
)
