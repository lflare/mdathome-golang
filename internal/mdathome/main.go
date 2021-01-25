package mdathome

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/gorilla/mux"
	"github.com/lflare/mdathome-golang/pkg/diskcache"
	"github.com/sirupsen/logrus"
)

var clientSettings = ClientSettings{
	CacheDirectory:            "cache/", // Default cache directory
	ClientPort:                443,      // Default to listen for requests on port 443
	OverridePortReport:        0,        // Default to advertise for port 443
	OverrideAddressReport:     "",       // Default to not overriding address report
	GracefulShutdownInSeconds: 300,      // Default 5m graceful shutdown

	MaxKilobitsPerSecond:       10000, // Default 10Mbps
	MaxCacheSizeInMebibytes:    10240, // Default 10GB
	MaxReportedSizeInMebibytes: 10240, // Default 10GB

	CacheScanIntervalInSeconds: 300,  // Default 5m scan interval
	CacheRefreshAgeInSeconds:   3600, // Default 1h cache refresh age
	MaxCacheScanTimeInSeconds:  15,   // Default 15s max scan period

	AllowHTTP2:              true,  // Allow HTTP2 by default
	AllowUpstreamPooling:    true,  // Allow upstream pooling by default
	AllowVisitorRefresh:     false, // Default to not allow visitors to force-refresh images through Cache-Control
	EnablePrometheusMetrics: false, // Default to not enable Prometheus metrics
	MaxMindLicenseKey:       "",    // Default to not have any MaxMind Geolocation DB
	OverrideUpstream:        "",    // Default to nil to follow upstream by controller
	RejectInvalidTokens:     true,  // Default to reject invalid tokens
	VerifyImageIntegrity:    false, // Default to not verify image integrity

	LogLevel:              "trace", // Default to "trace" for all logs
	MaxLogSizeInMebibytes: 64,      // Default to maximum log size of 64MiB
	MaxLogBackups:         3,       // Default to maximum log backups of 3
	MaxLogAgeInDays:       7,       // Default to maximum log age of 7 days
}
var serverResponse ServerResponse
var cache *diskcache.Cache
var timeLastRequest time.Time
var running = true
var client *http.Client

func requestHandler(w http.ResponseWriter, r *http.Request) {
	// Start timer
	startTime := time.Now()

	// Check if referer exists, else fake one
	if r.Header.Get("Referer") == "" {
		r.Header.Set("Referer", "None")
	}

	// Extract request variables
	tokens := mux.Vars(r)
	remoteAddr, _, _ := net.SplitHostPort(r.RemoteAddr)

	// Prepare logger for request
	requestLogger := log.WithFields(logrus.Fields{"url_path": r.URL.Path, "remote_addr": remoteAddr, "referer": r.Header.Get("Referer")})

	// Parse GeoIP
	labels := ""
	if geodb != nil {
		ip := net.ParseIP(remoteAddr)
		record, err := geodb.City(ip)
		if err == nil && record.Country.IsoCode != "" {
			labels = fmt.Sprintf(`{country=%q}`, record.Country.IsoCode)
		}
	}

	// Create all metric counters
	var (
		clientHitsTotal      = metrics.GetOrCreateCounter(fmt.Sprintf("client_hits_total%s", labels))
		clientMissedTotal    = metrics.GetOrCreateCounter(fmt.Sprintf("client_missed_total%s", labels))
		clientRefreshedTotal = metrics.GetOrCreateCounter(fmt.Sprintf("client_refreshed_total%s", labels))
		clientRequestsTotal  = metrics.GetOrCreateCounter(fmt.Sprintf("client_requests_total%s", labels))
		clientSkippedTotal   = metrics.GetOrCreateCounter(fmt.Sprintf("client_skipped_total%s", labels))

		clientDownloadedBytesTotal = metrics.GetOrCreateCounter(fmt.Sprintf("client_downloaded_bytes_total%s", labels))
		clientServedBytesTotal     = metrics.GetOrCreateCounter(fmt.Sprintf("client_served_bytes_total%s", labels))

		clientCorruptedTotal = metrics.GetOrCreateCounter(fmt.Sprintf("client_corrupted_total%s", labels))
		clientDroppedTotal   = metrics.GetOrCreateCounter(fmt.Sprintf("client_dropped_total%s", labels))
		clientFailedTotal    = metrics.GetOrCreateCounter(fmt.Sprintf("client_failed_total%s", labels))

		clientRequestDurationSeconds = metrics.GetOrCreateHistogram(fmt.Sprintf("client_request_duration_seconds%s", labels))
		clientRequestProcessSeconds  = metrics.GetOrCreateHistogram(fmt.Sprintf("client_request_process_seconds%s", labels))
	)

	// Sanitized URL
	if tokens["image_type"] != "data" && tokens["image_type"] != "data-saver" {
		requestLogger.WithFields(logrus.Fields{"event": "dropped", "reason": "invalid image type"}).Warnf("Request from %s dropped due to invalid image type", remoteAddr)
		clientDroppedTotal.Inc()
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if matched, _ := regexp.MatchString(`^[0-9a-f]{32}$`, tokens["chapter_hash"]); !matched {
		requestLogger.WithFields(logrus.Fields{"event": "dropped", "reason": "invalid url format"}).Warnf("Request from %s dropped due to invalid url format", remoteAddr)
		clientDroppedTotal.Inc()
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if matched, _ := regexp.MatchString(`^.+\.(jpg|jpeg|png|gif)$`, tokens["image_filename"]); !matched {
		requestLogger.WithFields(logrus.Fields{"event": "dropped", "reason": "invalid image extension"}).Warnf("Request from %s dropped due to invalid image extension", remoteAddr)
		clientDroppedTotal.Inc()
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// If configured to reject invalid tokens
	if clientSettings.RejectInvalidTokens {
		// Check if it is a test chapter
		if !isTestChapter(tokens["chapter_hash"]) {
			// Verify token if checking for invalid token and not a test chapter
			code, err := verifyToken(tokens["token"], tokens["chapter_hash"])
			if err != nil {
				requestLogger.WithFields(logrus.Fields{"event": "dropped", "reason": "invalid token"}).Warnf("Request from %s dropped due to invalid token", remoteAddr)
				w.WriteHeader(code)
				clientDroppedTotal.Inc()
				return
			}
		}
	}

	// Create sanitized url if everything checks out
	sanitizedURL := "/" + tokens["image_type"] + "/" + tokens["chapter_hash"] + "/" + tokens["image_filename"]

	// Update last request
	timeLastRequest = time.Now()

	// Properly handle MangaDex's Referer
	re := regexp.MustCompile(`https://mangadex.org/chapter/[0-9]+`)
	if matched := re.FindString(r.Header.Get("Referer")); matched != "" {
		r.Header.Set("Referer", matched)
	}

	// Add server headers
	serverHeader := fmt.Sprintf("MD@Home Golang Client %s (%d) - github.com/lflare/mdathome-golang", ClientVersion, ClientSpecification)
	w.Header().Set("Access-Control-Allow-Origin", "https://mangadex.org")
	w.Header().Set("Access-Control-Expose-Headers", "*")
	w.Header().Set("Cache-Control", "public, max-age=1209600")
	w.Header().Set("Server", serverHeader)
	w.Header().Set("Timing-Allow-Origin", "https://mangadex.org")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Log request
	requestLogger.WithFields(logrus.Fields{"event": "received"}).Infof("Request from %s received", remoteAddr)
	clientRequestsTotal.Inc()

	// Check if browser token exists
	if r.Header.Get("If-Modified-Since") != "" {
		// Log browser cache
		requestLogger.WithFields(logrus.Fields{"event": "cached"}).Debugf("Request from %s cached by browser", remoteAddr)
		clientSkippedTotal.Inc()
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Load cache
	imageFromCache, modTime, err := cache.Get(sanitizedURL)

	// Check image integrity if found in cache
	ok := (err == nil)
	if ok {
		// Check image content type
		contentType := http.DetectContentType(imageFromCache)
		if !strings.Contains(contentType, "image") {
			ok = false
		}

		// Check SHA256 hash if exists in URL (might be computationally heavy)
		if clientSettings.VerifyImageIntegrity && tokens["image_type"] == "data" {
			subTokens := strings.Split(tokens["image_filename"], "-")
			if len(subTokens) == 2 {
				// Check given hash length
				givenHash := strings.Split(subTokens[1], ".")[0]
				if len(givenHash) == 64 {
					// Calculate hash
					calculatedHash := fmt.Sprintf("%x", sha256.Sum256(imageFromCache))

					// Compare hash
					if givenHash != calculatedHash {
						requestLogger.WithFields(logrus.Fields{"event": "checksum", "given": givenHash, "calculated": calculatedHash}).Warnf("Request from %s generated invalid checksum %s != %s", calculatedHash, givenHash)
						clientCorruptedTotal.Inc()
						ok = false
					}
				}
			}
		}
	}

	// Check if image refresh is enabled and Cache-Control header is set
	if clientSettings.AllowVisitorRefresh && r.Header.Get("Cache-Control") == "no-cache" {
		// Log cache ignored
		requestLogger.WithFields(logrus.Fields{"event": "no-cache"}).Debugf("Request from %s ignored cache", remoteAddr)
		clientRefreshedTotal.Inc()

		// Set ok to false
		ok = false
	}

	// Check if image exists and is a proper image and if cache-control is set
	imageLength := 0
	if !ok {
		// Log cache miss
		requestLogger.WithFields(logrus.Fields{"event": "miss"}).Debugf("Request from %s missed cache", remoteAddr)
		clientMissedTotal.Inc()
		w.Header().Set("X-Cache", "MISS")

		// Send request
		imageFromUpstream, err := client.Get(serverResponse.ImageServer + sanitizedURL)
		if err != nil {
			requestLogger.WithFields(logrus.Fields{"event": "failed", "upstream": serverResponse.ImageServer + sanitizedURL, "error": err}).Warnf("Request from %s failed upstream: %v", remoteAddr, err)
			clientFailedTotal.Inc()
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		defer imageFromUpstream.Body.Close()

		// If not 200
		if imageFromUpstream.StatusCode != 200 {
			requestLogger.WithFields(logrus.Fields{"event": "failed", "error": "received non-200 status code", "status": imageFromUpstream.StatusCode}).Warnf("Request from %s failed upstream: %d", remoteAddr, imageFromUpstream.StatusCode)
			clientFailedTotal.Inc()
			w.WriteHeader(imageFromUpstream.StatusCode)
			return
		}

		// Set Content-Length if exists
		if contentLength := imageFromUpstream.Header.Get("Content-Length"); contentLength != "" {
			w.Header().Set("Content-Length", contentLength)
		}

		// Set Last-Modified
		modTime := time.Now()
		if lastModified := imageFromUpstream.Header.Get("Last-Modified"); lastModified != "" {
			w.Header().Set("Last-Modified", lastModified)

			if upstreamModTime, err := time.Parse(http.TimeFormat, lastModified); err == nil {
				modTime = upstreamModTime
			} else if seconds, err := strconv.Atoi(lastModified); err == nil && seconds > 0 {
				modTime = time.Unix(int64(seconds), 0)
			}
		}

		// Set timing header
		processedTime := time.Since(startTime).Milliseconds()
		requestLogger.WithFields(logrus.Fields{"event": "processed", "time_taken_ms": processedTime}).Tracef("Request from %s processed in %dms", remoteAddr, processedTime)
		clientRequestProcessSeconds.Update(float64(processedTime) / 1000.0)
		w.Header().Set("X-Time-Taken", strconv.Itoa(int(processedTime)))

		// Copy request to response body
		var imageBuffer bytes.Buffer
		_, err = io.Copy(w, io.TeeReader(imageFromUpstream.Body, &imageBuffer))

		// Check if image was streamed properly
		if err != nil {
			requestLogger.WithFields(logrus.Fields{"event": "failed", "upstream": serverResponse.ImageServer + sanitizedURL, "error": err}).Warnf("Request from %s failed downstream: %v", remoteAddr, err)
			clientFailedTotal.Inc()
			return
		}

		// Save hash
		err = cache.Set(sanitizedURL, modTime, imageBuffer.Bytes())
		if err != nil {
			requestLogger.WithFields(logrus.Fields{"event": "failed", "error": err}).Warnf("Request from %s failed to save: %v", remoteAddr, err)
			clientFailedTotal.Inc()
		}

		// Update bytes downloaded
		imageLength = len(imageBuffer.Bytes())
		clientDownloadedBytesTotal.Add(imageLength)
		requestLogger.WithFields(logrus.Fields{"event": "committed", "image_length": imageLength}).Debug("Request from %s committed with size %d bytes", imageLength)
	} else {
		// Get length
		imageLength = len(imageFromCache)

		// Log cache hit
		requestLogger.WithFields(logrus.Fields{"event": "hit"}).Debugf("Request from %s hit cache", remoteAddr)
		clientHitsTotal.Inc()
		w.Header().Set("X-Cache", "HIT")

		// Set Content-Length & Last-Modified
		w.Header().Set("Content-Length", strconv.Itoa(imageLength))
		w.Header().Set("Last-Modified", modTime.Format(http.TimeFormat))

		// Set timing header
		processedTime := time.Since(startTime).Milliseconds()
		requestLogger.WithFields(logrus.Fields{"event": "processed", "time_taken_ms": processedTime}).Tracef("Request from %s processed in %dms", remoteAddr, processedTime)
		clientRequestProcessSeconds.Update(float64(processedTime) / 1000.0)
		w.Header().Set("X-Time-Taken", strconv.Itoa(int(processedTime)))

		// Convert bytes object into reader and send to client
		imageReader := bytes.NewReader(imageFromCache)
		_, err := io.Copy(w, imageReader)

		// Check if image was streamed properly
		if err != nil {
			requestLogger.WithFields(logrus.Fields{"event": "failed", "upstream": serverResponse.ImageServer + sanitizedURL, "error": err}).Warnf("Request from %s failed downstream: %v", remoteAddr, err)
			clientFailedTotal.Inc()
			return
		}
	}

	// End time
	totalTime := time.Since(startTime).Milliseconds()
	requestLogger.WithFields(logrus.Fields{"event": "completed", "time_taken_ms": totalTime, "image_length": imageLength}).Tracef("Request from %s completed in %dms and %d bytes", remoteAddr, totalTime, imageLength)
	clientRequestDurationSeconds.Update(float64(totalTime) / 1000.0)
	w.Header().Set("X-Time-Taken", strconv.Itoa(int(totalTime)))

	// Update bytes served to readers
	clientServedBytesTotal.Add(imageLength)
}

// ShrinkDatabase initialises and shrinks the MD@Home database
func ShrinkDatabase() {
	// Load & prepare client settings
	loadClientSettings()
	saveClientSettings()

	// Prepare diskcache
	log.Println("Preparing database...")
	cache = diskcache.New(clientSettings.CacheDirectory, 0, 0, 0, 0, log, clientCacheSize, clientCacheLimit)
	defer cache.Close()

	// Attempts to start cache shrinking
	log.Println("Shrinking database...")
	cache.ShrinkDatabase()
}

// StartServer starts the MD@Home client
func StartServer() {
	// Check client version
	checkClientVersion()

	// Load & prepare client settings
	loadClientSettings()
	saveClientSettings()

	// Initialise logger
	initLogger(clientSettings.LogLevel, clientSettings.MaxLogSizeInMebibytes, clientSettings.MaxLogBackups, clientSettings.MaxLogAgeInDays)

	// Prepare diskcache
	cache = diskcache.New(
		clientSettings.CacheDirectory,
		clientSettings.MaxCacheSizeInMebibytes*1024*1024,
		clientSettings.CacheScanIntervalInSeconds,
		clientSettings.CacheRefreshAgeInSeconds,
		clientSettings.MaxCacheScanTimeInSeconds,
		log,
		clientCacheSize,
		clientCacheLimit,
	)
	defer cache.Close()

	// Prepare geoip
	if clientSettings.MaxMindLicenseKey != "" {
		log.Warnf("Loading geolocation data in the background...")
		go prepareGeoIPDatabase()
		defer geodb.Close()
	}

	// Prepare upstream client
	tr := &http.Transport{
		MaxIdleConns:      10,
		IdleConnTimeout:   60 * time.Second,
		DisableKeepAlives: !clientSettings.AllowUpstreamPooling,
	}
	client = &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}

	// Register shutdown handler
	serverShutdownHandler()

	// Retrieve TLS certificates
	serverResponse = *backendPing()
	if serverResponse.TLS.Certificate == "" {
		log.Fatalln("Unable to contact API server!")
	}

	// Attempt to parse TLS data
	keyPair, err := tls.X509KeyPair([]byte(serverResponse.TLS.Certificate), []byte(serverResponse.TLS.PrivateKey))
	if err != nil {
		log.Fatalf("Cannot parse TLS data %v - %v", serverResponse, err)
	}

	// Start background worker
	go backgroundWorker()

	// Prepare server
	r := mux.NewRouter()
	if clientSettings.EnablePrometheusMetrics {
		r.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
			metrics.WritePrometheus(w, true)
		})
	}

	// Prepare paths
	r.HandleFunc("/{image_type}/{chapter_hash}/{image_filename}", requestHandler)
	r.HandleFunc("/{token}/{image_type}/{chapter_hash}/{image_filename}", requestHandler)

	http.Handle("/", r)

	// Start proxy server
	err = listenAndServeTLSKeyPair(":"+strconv.Itoa(clientSettings.ClientPort), clientSettings.AllowHTTP2, keyPair, r)
	if err != nil {
		log.Fatalf("Cannot start server: %v", err)
	}
}
