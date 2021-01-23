package mdathome

import (
	"github.com/VictoriaMetrics/metrics"
)

var (
	clientRequestDurationSeconds = metrics.NewHistogram("client_request_duration_seconds")
	clientRequestProcessSeconds  = metrics.NewHistogram("client_request_process_seconds")

	clientCacheSize  = metrics.NewCounter("client_cache_size")
	clientCacheLimit = metrics.NewCounter("client_cache_limit")
)
