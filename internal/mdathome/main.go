package mdathome

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var serverResponse ServerResponse
var cache *Cache
var timeLastRequest time.Time
var running = true
var client *http.Client
var certHandler *certificateHandler

var clientHostname string

func requestHandler(w http.ResponseWriter, r *http.Request) {
	// Start timer
	startTime := time.Now()

	// Check if referer exists, else fake one
	if r.Header.Get("Referer") == "" {
		r.Header.Set("Referer", "None")
	}

	// Extract request variables
	tokens := mux.Vars(r)
	remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteAddr = r.RemoteAddr
	}

	// Prepare logger for request
	requestLogger := log.WithFields(logrus.Fields{"type": "request", "url_path": r.URL.Path, "remote_addr": remoteAddr, "referer": r.Header.Get("Referer")})

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

	// Check if hostname is rejected
	requestHostname := strings.Split(r.Host, ":")[0]
	if viper.GetBool("security.reject_invalid_hostname") && requestHostname != clientHostname {
		requestLogger.WithFields(logrus.Fields{"event": "dropped", "reason": "invalid hostname"}).Warnf("Request from %s dropped due to invalid hostname: %s", remoteAddr, requestHostname)
		clientDroppedTotal.Inc()
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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
	if viper.GetBool("security.reject_invalid_tokens") && !serverResponse.DisableTokens {
		// Verify token if checking for invalid token and not a test chapter
		if code, err := verifyToken(tokens["token"], tokens["chapter_hash"]); err != nil {
			requestLogger.WithFields(logrus.Fields{"event": "dropped", "reason": "invalid token"}).Warnf("Request from %s dropped due to invalid token", remoteAddr)
			clientDroppedTotal.Inc()
			w.WriteHeader(code)
			return
		}
	}

	// Create sanitized url if everything checks out
	sanitizedURL := "/" + tokens["image_type"] + "/" + tokens["chapter_hash"] + "/" + tokens["image_filename"]

	// Update requestLogger with new fields
	requestLogger = requestLogger.WithFields(logrus.Fields{
		"url_path":     r.URL.Path,
		"remote_addr":  remoteAddr,
		"referer":      r.Header.Get("Referer"),
		"token":        tokens["token"],
		"image_type":   tokens["image_type"],
		"chapter_hash": tokens["chapter_hash"],
		"filename":     tokens["image_filename"],
	})

	// Update last request
	timeLastRequest = time.Now()

	// Properly handle MangaDex's Referer
	re := regexp.MustCompile(`https://mangadex.org/chapter/[0-9]+`)
	if matched := re.FindString(r.Header.Get("Referer")); matched != "" {
		r.Header.Set("Referer", matched)
	}

	// Add server headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "*")
	w.Header().Set("Cache-Control", "public, max-age=1209600")
	w.Header().Set("Timing-Allow-Origin", "*")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Depending on client configuration, choose to hide Server header identifier
	if viper.GetBool("security.send_server_header") {
		serverHeader := fmt.Sprintf("MD@Home Golang Client %s (%d) - github.com/lflare/mdathome-golang", ClientVersion, ClientSpecification)
		w.Header().Set("Server", serverHeader)
	}

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

	// Load image from cache
	imageFile, imageSize, imageModTime, err := cache.Get(sanitizedURL)

	// Check image integrity if found in cache
	var imageBuffer bytes.Buffer
	imageOk := (err == nil)
	if imageOk {
		// Close image file at the end of goroutine
		defer imageFile.Close()

		// Check if client is running in low-memory mode
		if !viper.GetBool("performance.low_memory_mode") {
			// Load image from disk to buffer if not low-memory mode
			imageBuffer.Grow(int(imageSize))
			if _, err := io.Copy(&imageBuffer, imageFile); err != nil {
				requestLogger.Errorf("Failed to copy image to buffer: %v", err)
			}

			// Check if verifying image integrity
			if viper.GetBool("security.verify_image_integrity") && tokens["image_type"] == "data" {
				// Check and get hash from image filename
				subTokens := strings.Split(tokens["image_filename"], "-")
				if len(subTokens) == 2 {
					// Check and get given hash length
					givenHash := strings.Split(subTokens[1], ".")[0]
					if len(givenHash) == 64 {
						// Calculate actual image hash
						calculatedHash := fmt.Sprintf("%x", sha256.Sum256(imageBuffer.Bytes()))

						// Compare hash
						if givenHash != calculatedHash {
							// Log cache corrupted
							requestLogger.WithFields(logrus.Fields{"event": "checksum", "given": givenHash, "calculated": calculatedHash}).Warnf("Request from %s generated invalid checksum %s != %s", remoteAddr, calculatedHash, givenHash)
							clientCorruptedTotal.Inc()

							// Set imageOk to false
							imageOk = false
						}
					}
				}
			}
		}
	}

	// Check if image refresh is enabled and Cache-Control header is set
	if viper.GetBool("security.allow_visitor_cache_refresh") && r.Header.Get("Cache-Control") == "no-cache" {
		// Log cache ignored
		requestLogger.WithFields(logrus.Fields{"event": "no-cache"}).Debugf("Request from %s ignored cache", remoteAddr)
		clientRefreshedTotal.Inc()

		// Set imageOk to false
		imageOk = false
	}

	// Check if image exists and is a proper image and if cache-control is set
	imageLength := 0
	if !imageOk {
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
		requestLogger.WithFields(logrus.Fields{"event": "committed", "image_length": imageLength}).Debugf("Request from %s committed with size %d bytes", remoteAddr, imageLength)
	} else {
		// Get length
		imageLength = int(imageSize)

		// Log cache hit
		requestLogger.WithFields(logrus.Fields{"event": "hit"}).Debugf("Request from %s hit cache", remoteAddr)
		clientHitsTotal.Inc()
		w.Header().Set("X-Cache", "HIT")

		// Set Content-Length & Last-Modified
		w.Header().Set("Content-Length", strconv.Itoa(imageLength))
		w.Header().Set("Last-Modified", imageModTime.Format(http.TimeFormat))

		// Set timing header
		processedTime := time.Since(startTime).Milliseconds()
		requestLogger.WithFields(logrus.Fields{"event": "processed", "time_taken_ms": processedTime}).Tracef("Request from %s processed in %dms", remoteAddr, processedTime)
		clientRequestProcessSeconds.Update(float64(processedTime) / 1000.0)
		w.Header().Set("X-Time-Taken", strconv.Itoa(int(processedTime)))

		// Stream image to client
		var err error
		if imageBuffer.Len() == 0 {
			_, err = io.Copy(w, imageFile)
		} else {
			imageReader := bytes.NewReader(imageBuffer.Bytes())
			_, err = io.Copy(w, imageReader)
		}

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
	// Prepare diskcache
	log.Info("Preparing database...")
	cache = OpenCache(viper.GetString(KeyCacheDirectory), 0)
	defer cache.Close()

	// Attempts to start cache shrinking
	log.Info("Shrinking database...")
	if err := cache.Shrink(); err != nil {
		log.Errorf("Failed to shrink database: %v", err)
	}
}

// StartServer starts the MD@Home client
func StartServer() {
	// Watch for configuration changes
	prepareConfiguration()

	// Check client version
	checkClientVersion()

	// Initialise logger
	initLogger(viper.GetString("log.level"), viper.GetInt("log.max_size_mebibytes"), viper.GetInt("log.max_backups"), viper.GetInt("log.max_age_days"))

	// Prepare diskcache
	cache = OpenCache(
		viper.GetString(KeyCacheDirectory),
		viper.GetInt(KeyCacheSize)*1024*1024,
	)
	defer cache.Close()

	// Prepare MaxMind geolocation database
	if viper.GetString("metrics.maxmind_license_key") != "" {
		log.Warnf("Loading geolocation data in the background...")
		go prepareGeoIPDatabase()
		defer geodb.Close()
	}

	// Prepare transport
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 100
	transport.MaxConnsPerHost = 0
	transport.IdleConnTimeout = 60 * time.Second
	transport.DisableKeepAlives = !viper.GetBool("performance.upstream_connection_reuse")

	// Prepare upstream client
	client = &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Register shutdown handler
	registerShutdownHandler()

	// Prepare TLS reloader
	certHandler = NewCertificateReloader(controlGetCertificate())
	go func() {
		for {
			time.Sleep(24 * time.Hour)

			// Update certificate
			log.Infof("Reloading certificates...")
			if err := certHandler.updateCertificate(controlGetCertificate()); err != nil {
				log.Errorf("Failed to reload certificate: %v", err)
			}
		}
	}()

	// Start background worker
	go startBackgroundWorker()

	// Prepare router
	r := mux.NewRouter()

	// Prepare paths
	r.HandleFunc("/{image_type}/{chapter_hash}/{image_filename}", requestHandler)
	r.HandleFunc("/{token}/{image_type}/{chapter_hash}/{image_filename}", requestHandler)

	// Add robots.txt
	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("User-Agent: *\nDisallow: /\n")); err != nil {
			log.Errorf("Failed to write robots.txt: %v", err)
		}
	})

	// Handle Prometheus metrics
	r.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		if viper.GetBool("metrics.enable_prometheus") {
			metrics.WritePrometheus(w, true)
		}
	})

	// If configured behind reverse proxies
	if viper.GetBool("metrics.use_forwarded_for_headers") {
		r.Use(handlers.ProxyHeaders)
	}

	// Set router
	http.Handle("/", handlers.RecoveryHandler()(handlers.CompressHandler(r)))

	// Start server
	err := listenAndServeTLSKeyPair(r)
	if err != nil {
		log.Fatalf("Cannot start server: %v", err)
	}
}
