package mdathome

import (
	"github.com/VictoriaMetrics/metrics"
)

var (
	clientCacheSize  = metrics.NewCounter("client_cache_size")
	clientCacheLimit = metrics.NewCounter("client_cache_limit")
)
