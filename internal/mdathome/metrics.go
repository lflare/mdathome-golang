package mdathome

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	prometheusRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "requests",
		Help: "Total number of all requests",
	})
	prometheusHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hit",
		Help: "Total number of cache-hit requests",
	})
	prometheusMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "miss",
		Help: "Total number of cache-miss requests",
	})
	prometheusDropped = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dropped",
		Help: "Total number of dropped requests",
	})
	prometheusCached = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cached",
		Help: "Total number of browser-cached requests",
	})
	prometheusForced = promauto.NewCounter(prometheus.CounterOpts{
		Name: "forced",
		Help: "Total number of no-cache requests",
	})
	prometheusFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "failed",
		Help: "Total number of failed requests",
	})
	prometheusProcessedTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "processed_times",
		Help:    "Processed times of requests in milliseconds",
		Buckets: prometheus.ExponentialBuckets(1, 1.25, 50),
	})
	prometheusCompletionTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "completion_times",
		Help:    "Completion times of requests in milliseconds",
		Buckets: prometheus.ExponentialBuckets(1, 1.25, 50),
	})
)
