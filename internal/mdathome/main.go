package mdathome

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
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
	ClientPort:                44300,    // Default client port
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
	OverrideUpstream:        "",    // Default to nil to follow upstream by controller
	RejectInvalidTokens:     false, // Default to not reject invalid tokens
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
	remoteAddr := r.RemoteAddr

	// Prepare logger for request
	requestLogger := log.WithFields(logrus.Fields{"url_path": r.URL.Path, "remote_addr": remoteAddr, "referer": r.Header.Get("Referer")})

	// Sanitized URL
	if tokens["image_type"] != "data" && tokens["image_type"] != "data-saver" {
		requestLogger.WithFields(logrus.Fields{"event": "dropped", "reason": "invalid image type"}).Warnf("Request from %s dropped due to invalid image type", remoteAddr)
		prometheusDropped.Inc()
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if matched, _ := regexp.MatchString(`^[0-9a-f]{32}$`, tokens["chapter_hash"]); !matched {
		requestLogger.WithFields(logrus.Fields{"event": "dropped", "reason": "invalid url format"}).Warnf("Request from %s dropped due to invalid url format", remoteAddr)
		prometheusDropped.Inc()
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if matched, _ := regexp.MatchString(`^.+\.(jpg|jpeg|png|gif)$`, tokens["image_filename"]); !matched {
		requestLogger.WithFields(logrus.Fields{"event": "dropped", "reason": "invalid image extension"}).Warnf("Request from %s dropped due to invalid image extension", remoteAddr)
		prometheusDropped.Inc()
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if token is valid
	code, err := verifyToken(tokens["token"], tokens["chapter_hash"])
	if err != nil {
		// Reject invalid tokens if we enabled it
		if clientSettings.RejectInvalidTokens {
			requestLogger.WithFields(logrus.Fields{"event": "dropped", "reason": "invalid token"}).Warnf("Request from %s dropped due to invalid token", remoteAddr)
			w.WriteHeader(code)
			prometheusDropped.Inc()
			return
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
	serverHeader := fmt.Sprintf("MD@Home Golang Client %s (%d) - github.com/lflare/mdathome-golang", clientVersion, specVersion)
	w.Header().Set("Access-Control-Allow-Origin", "https://mangadex.org")
	w.Header().Set("Access-Control-Expose-Headers", "*")
	w.Header().Set("Cache-Control", "public, max-age=1209600")
	w.Header().Set("Server", serverHeader)
	w.Header().Set("Timing-Allow-Origin", "https://mangadex.org")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Log request
	requestLogger.WithFields(logrus.Fields{"event": "received"}).Infof("Request from %s received", remoteAddr)
	prometheusRequests.Inc()

	// Check if browser token exists
	if r.Header.Get("If-Modified-Since") != "" {
		// Log browser cache
		requestLogger.WithFields(logrus.Fields{"event": "cached"}).Debugf("Request from %s cached by browser", remoteAddr)
		prometheusCached.Inc()
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Load cache
	imageFromCache, err := cache.Get(sanitizedURL)

	// Check image integrity if found in cache
	ok := (err == nil)
	if ok {
		// Check image content type
		contentType := http.DetectContentType(imageFromCache)
		if !strings.Contains(contentType, "image") {
			ok = false
		}

		// Check SHA256 hash if exists in URL (might be computationally heavy)
		if clientSettings.VerifyImageIntegrity {
			subTokens := strings.Split(tokens["image_filename"], "-")
			if len(subTokens) == 2 {
				// Check given hash length
				givenHash := strings.Split(subTokens[1], ".")[0]
				if len(givenHash) == 64 {
					// Calculate hash
					calculatedHash := fmt.Sprintf("%x", sha256.Sum256(imageFromCache))

					// Compare hash
					if givenHash != calculatedHash {
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
		prometheusForced.Inc()

		// Set ok to false
		ok = false
	}

	// Check if image exists and is a proper image and if cache-control is set
	if !ok {
		// Log cache miss
		requestLogger.WithFields(logrus.Fields{"event": "miss"}).Debugf("Request from %s missed cache", remoteAddr)
		prometheusMiss.Inc()
		w.Header().Set("X-Cache", "MISS")

		// Send request
		imageFromUpstream, err := client.Get(serverResponse.ImageServer + sanitizedURL)
		if err != nil {
			requestLogger.WithFields(logrus.Fields{"event": "failed", "upstream": serverResponse.ImageServer + sanitizedURL, "error": err}).Warnf("Request from %s failed upstream: %v", remoteAddr, err)
			prometheusFailed.Inc()
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		defer imageFromUpstream.Body.Close()

		// If not 200
		if imageFromUpstream.StatusCode != 200 {
			requestLogger.WithFields(logrus.Fields{"event": "failed", "error": "received non-200 status code", "status": imageFromUpstream.StatusCode}).Warnf("Request from %s failed upstream: %d", remoteAddr, imageFromUpstream.StatusCode)
			prometheusFailed.Inc()
			w.WriteHeader(imageFromUpstream.StatusCode)
			return
		}

		// Set Content-Length
		w.Header().Set("Content-Length", imageFromUpstream.Header.Get("Content-Length"))

		// Set timing header
		processedTime := time.Since(startTime).Milliseconds()
		requestLogger.WithFields(logrus.Fields{"event": "processed", "time_taken": processedTime}).Tracef("Request from %s processed in %dms", remoteAddr, processedTime)
		prometheusProcessedTime.Update(float64(processedTime))
		w.Header().Set("X-Time-Taken", strconv.Itoa(int(processedTime)))

		// Copy request to response body
		var imageBuffer bytes.Buffer
		_, err = io.Copy(w, io.TeeReader(imageFromUpstream.Body, &imageBuffer))

		// Check if image was streamed properly
		if err != nil {
			requestLogger.WithFields(logrus.Fields{"event": "failed", "upstream": serverResponse.ImageServer + sanitizedURL, "error": err}).Warnf("Request from %s failed downstream: %v", remoteAddr, err)
			prometheusFailed.Inc()
			return
		}

		// Save hash
		err = cache.Set(sanitizedURL, imageBuffer.Bytes())
		if err != nil {
			requestLogger.WithFields(logrus.Fields{"event": "failed", "error": err}).Warnf("Request from %s failed to save: %v", remoteAddr, err)
			prometheusFailed.Inc()
		}
	} else {
		// Get length
		length := len(imageFromCache)
		image := make([]byte, length)
		copy(image, imageFromCache)

		// Log cache hit
		requestLogger.WithFields(logrus.Fields{"event": "hit"}).Debugf("Request from %s hit cache", remoteAddr)
		prometheusHit.Inc()
		w.Header().Set("X-Cache", "HIT")

		// Set Content-Length
		w.Header().Set("Content-Length", strconv.Itoa(length))

		// Set timing header
		processedTime := time.Since(startTime).Milliseconds()
		requestLogger.WithFields(logrus.Fields{"event": "processed", "time_taken": processedTime}).Tracef("Request from %s processed in %dms", remoteAddr, processedTime)
		prometheusProcessedTime.Update(float64(processedTime))
		w.Header().Set("X-Time-Taken", strconv.Itoa(int(processedTime)))

		// Convert bytes object into reader and send to client
		imageReader := bytes.NewReader(image)
		_, err := io.Copy(w, imageReader)

		// Check if image was streamed properly
		if err != nil {
			requestLogger.WithFields(logrus.Fields{"event": "failed", "upstream": serverResponse.ImageServer + sanitizedURL, "error": err}).Warnf("Request from %s failed downstream: %v", remoteAddr, err)
			prometheusFailed.Inc()
			return
		}
	}

	// End time
	totalTime := time.Since(startTime).Milliseconds()
	requestLogger.WithFields(logrus.Fields{"event": "completed", "time_taken": totalTime}).Tracef("Request from %s completed in %dms", remoteAddr, totalTime)
	prometheusCompletionTime.Update(float64(totalTime))
	w.Header().Set("X-Time-Taken", strconv.Itoa(int(totalTime)))
}

// ShrinkDatabase initialises and shrinks the MD@Home database
func ShrinkDatabase() {
	// Load & prepare client settings
	loadClientSettings()
	saveClientSettings()

	// Prepare diskcache
	log.Println("Preparing database...")
	cache = diskcache.New(clientSettings.CacheDirectory, 0, 0, 0, 0)
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
	)
	defer cache.Close()

	// Prepare upstream client
	tr := &http.Transport{
		MaxIdleConns:      10,
		IdleConnTimeout:   60 * time.Second,
		DisableKeepAlives: !clientSettings.AllowUpstreamPooling,
	}
	client = &http.Client{Transport: tr}

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
	r.HandleFunc("/{image_type}/{chapter_hash}/{image_filename}", requestHandler)
	r.HandleFunc("/{token}/{image_type}/{chapter_hash}/{image_filename}", requestHandler)
	http.Handle("/", r)

	// Start proxy server
	err = listenAndServeTLSKeyPair(":"+strconv.Itoa(clientSettings.ClientPort), clientSettings.AllowHTTP2, keyPair, r)
	if err != nil {
		log.Fatalf("Cannot start server: %v", err)
	}
}
