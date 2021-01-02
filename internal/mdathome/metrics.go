package mdathome

import (
	"github.com/VictoriaMetrics/metrics"
)

var (
	prometheusRequests       = metrics.NewCounter("requests")
	prometheusHit            = metrics.NewCounter("hit")
	prometheusMiss           = metrics.NewCounter("miss")
	prometheusDropped        = metrics.NewCounter("dropped")
	prometheusCached         = metrics.NewCounter("cached")
	prometheusForced         = metrics.NewCounter("forced")
	prometheusFailed         = metrics.NewCounter("failed")
	prometheusProcessedTime  = metrics.NewHistogram("processed_times")
	prometheusCompletionTime = metrics.NewHistogram("completion_times")
)
