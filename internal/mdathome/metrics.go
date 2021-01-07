package mdathome

import (
	"github.com/VictoriaMetrics/metrics"
)

var (
	clientHitsTotal      = metrics.NewCounter("client_hits_total")
	clientMissedTotal    = metrics.NewCounter("client_missed_total")
	clientRefreshedTotal = metrics.NewCounter("client_refreshed_total")
	clientRequestsTotal  = metrics.NewCounter("client_requests_total")
	clientSkippedTotal   = metrics.NewCounter("client_skipped_total")

	clientDownloadedBytesTotal = metrics.NewCounter("client_downloaded_bytes_total")
	clientServedBytesTotal     = metrics.NewCounter("client_served_bytes_total")

	clientCorruptedTotal = metrics.NewCounter("client_corrupted_total")
	clientDroppedTotal   = metrics.NewCounter("client_dropped_total")
	clientFailedTotal    = metrics.NewCounter("client_failed_total")

	clientRequestDurationSeconds = metrics.NewHistogram("client_request_duration_seconds")
	clientRequestProcessSeconds  = metrics.NewHistogram("client_request_process_seconds")
)
