package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/lflare/diskcache-golang"
	"golang.org/x/crypto/nacl/box"
)

// Global variables
var clientSettings = ClientSettings{
	CacheDirectory:             "cache/", // Default cache directory
	ClientPort:                 44300,    // Default client port
	MaxKilobitsPerSecond:       10000,    // Default 10Mbps
	MaxCacheSizeInMebibytes:    10240,    // Default 10GB
	MaxReportedSizeInMebibytes: 10240,    // Default 10GB
	GracefulShutdownInSeconds:  300,      // Default 5m graceful shutdown
	CacheScanIntervalInSeconds: 300,      // Default 5m scan interval
	MaxCacheScanTimeInSeconds:  15,       // Default 15s max scan period
	RejectInvalidTokens:        false,    // Default to not reject invalid tokens
}
var serverResponse ServerResponse
var cache *diskcache.Cache
var timeLastRequest time.Time
var running = true
var client *http.Client

// Swap the following for backend testing
// var apiBackend := "https://mangadex-test.net"
var apiBackend = "https://api.mangadex.network"

func saveClientSettings() {
	clientSettingsSampleBytes, err := json.MarshalIndent(clientSettings, "", "    ")
	if err != nil {
		log.Fatalln("Failed to marshal sample settings.json")
	}

	err = ioutil.WriteFile("settings.json", clientSettingsSampleBytes, 0600)
	if err != nil {
		log.Fatalf("Failed to create sample settings.json: %v", err)
	}
}

// Client setting handler
func loadClientSettings() {
	// Read JSON from file
	clientSettingsJson, err := ioutil.ReadFile("settings.json")
	if err != nil {
		log.Printf("Failed to read client configuration file - %v", err)
		saveClientSettings()
		log.Fatalf("Created sample settings.json! Please edit it before running again!")
	}

	// Unmarshal JSON to clientSettings struct
	err = json.Unmarshal(clientSettingsJson, &clientSettings)
	if err != nil {
		log.Fatalf("Unable to unmarshal JSON file: %v", err)
	}

	// Check client configuration
	if clientSettings.ClientSecret == "" {
		log.Fatalf("Empty secret! Cannot run!")
	}

	// Print client configuration
	log.Printf("Client configuration loaded: %+v", clientSettings)
}

// Server ping handler
func pingServer() *ServerResponse {
	// Create settings JSON
	settings := ServerSettings{
		Secret:       clientSettings.ClientSecret,
		Port:         clientSettings.ClientPort,
		DiskSpace:    clientSettings.MaxCacheSizeInMebibytes * 1024 * 1024, // 1GB
		NetworkSpeed: clientSettings.MaxKilobitsPerSecond * 1000 / 8,       // 100Mbps
		BuildVersion: 13,
		TlsCreatedAt: nil,
	}
	settingsJson, _ := json.Marshal(&settings)

	// Ping backend server
	r, err := http.Post(apiBackend+"/ping", "application/json", bytes.NewBuffer(settingsJson))
	if err != nil {
		log.Printf("Failed to ping control server: %v", err)
		return nil
	}
	defer r.Body.Close()

	// Read response fully
	response, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to ping control server: %v", err)
		return nil
	}

	// Print server settings out
	printableResponse := string(response)
	tlsIndex := strings.Index(printableResponse, "\"tls\"")
	if tlsIndex == -1 {
		log.Printf("Received invalid server response: %s", printableResponse)

		if serverResponse.Tls.Certificate == "" {
			log.Fatalln("No valid TLS certificate found in memory, cannot continue!")
		}
		return nil
	}
	log.Printf("Server settings received! - %s...", string(response[:tlsIndex]))

	// Decode & unmarshal server response
	newServerResponse := ServerResponse{}
	err = json.Unmarshal(response, &newServerResponse)
	if err != nil {
		log.Printf("Failed to ping control server: %v", err)
		return nil
	}

	// Check struct
	if newServerResponse.ImageServer == "" {
		log.Printf("Failed to verify server response: %s", response)
		return nil
	}

	// Return server response
	return &newServerResponse
}

// Server ping loop handler
func BackgroundLoop() {
	// Wait 15 seconds
	log.Println("Starting background jobs!")
	time.Sleep(15 * time.Second)

	for running == true {
		// Reload client configuration
		log.Println("Reloading client configuration")
		loadClientSettings()

		// Update max cache size
		cache.UpdateCacheLimit(clientSettings.MaxCacheSizeInMebibytes * 1024 * 1024)
		cache.UpdateCacheScanInterval(clientSettings.CacheScanIntervalInSeconds)

		// Update server response in a goroutine
		newServerResponse := pingServer()
		if newServerResponse != nil {
			serverResponse = *newServerResponse
		}

		// Wait 15 seconds
		time.Sleep(15 * time.Second)
	}
}

func VerifyToken(tokenString string, chapterHash string) (error, int) {
	// Check if given token string is empty
	if tokenString == "" {
		return fmt.Errorf("Token is empty!"), 403
	}

	// Decode base64-encoded token & key
	tokenBytes, err := base64.RawURLEncoding.DecodeString(tokenString)
	if err != nil {
		return fmt.Errorf("Cannot decode token - %v", err), 403
	}
	keyBytes, err := base64.StdEncoding.DecodeString(serverResponse.TokenKey)
	if err != nil {
		return fmt.Errorf("Cannot decode key - %v", err), 403
	}

	// Copy over byte slices to fixed-length byte arrays for decryption
	var nonce [24]byte
	copy(nonce[:], tokenBytes[:24])
	var key [32]byte
	copy(key[:], keyBytes[:32])

	// Decrypt token
	data, ok := box.OpenAfterPrecomputation(nil, tokenBytes[24:], &nonce, &key)
	if !ok {
		return fmt.Errorf("Failed to decrypt token!"), 403
	}

	// Unmarshal to struct
	token := Token{}
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf("Failed to unmarshal token - %v", err), 403
	}

	// Parse expiry time
	expires, err := time.Parse(time.RFC3339, token.Expires)
	if err != nil {
		return fmt.Errorf("Failed to parse expiry from token - %v", err), 403
	}

	// Check token expiry timing
	if time.Now().After(expires) {
		return fmt.Errorf("Token has expired"), 403
	}

	// Check that chapter hashes are the same
	if token.Hash != chapterHash {
		return fmt.Errorf("Token hash invalid"), 410
	}

	// Token is valid
	return nil, 0
}

// Image handler
func RequestHandler(w http.ResponseWriter, r *http.Request) {
	// Start timer
	startTime := time.Now()

	// Extract tokens
	tokens := mux.Vars(r)

	// Sanitized URL
	if tokens["image_type"] != "data" && tokens["image_type"] != "data-saver" {
		log.Printf("Request for %s failed", r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if matched, _ := regexp.MatchString(`^[0-9a-f]{32}$`, tokens["chapter_hash"]); !matched {
		log.Printf("Request for %s failed", r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if matched, _ := regexp.MatchString(`[a-zA-Z0-9]{1,4}\.(jpg|jpeg|png|gif)$`, tokens["image_filename"]); !matched {
		log.Printf("Request for %s failed", r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if token is valid
	err, code := VerifyToken(tokens["token"], tokens["chapter_hash"])
	if err != nil {
		log.Printf("Request for %s invalid - %v", r.URL.Path, err)

		// Reject invalid tokens if we enabled it
		if clientSettings.RejectInvalidTokens {
			w.WriteHeader(code)
			return
		}
	}

	// Create sanitized url if everything checks out
	sanitizedUrl := "/" + tokens["image_type"] + "/" + tokens["chapter_hash"] + "/" + tokens["image_filename"]

	// Update last request
	timeLastRequest = time.Now()

	// Check if referer exists, else fake one
	if r.Header.Get("Referer") == "" {
		r.Header.Set("Referer", "None")
	}

	// Properly handle MangaDex's Referer
	re := regexp.MustCompile(`https://mangadex.org/chapter/[0-9]+`)
	if matched := re.FindString(r.Header.Get("Referer")); matched != "" {
		r.Header.Set("Referer", matched)
	}

	// Add server headers
	w.Header().Set("Access-Control-Allow-Origin", "https://mangadex.org")
	w.Header().Set("Access-Control-Expose-Headers", "*")
	w.Header().Set("Cache-Control", "public, max-age=1209600")
	w.Header().Set("Server", "MangaDex@Home - github.com/lflare/mdathome-golang")
	w.Header().Set("Timing-Allow-Origin", "https://mangadex.org")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Log request
	log.Printf("Request for %s - %s - %s received", sanitizedUrl, r.RemoteAddr, r.Header.Get("Referer"))

	// Check if browser token exists
	if r.Header.Get("If-Modified-Since") != "" {
		// Log browser cache
		log.Printf("Request for %s - %s - %s cached by browser", sanitizedUrl, r.RemoteAddr, r.Header.Get("Referer"))
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Check if image already in cache or if cache-control is set
	if imageFromCache, ok := cache.Get(sanitizedUrl); !ok || r.Header.Get("Cache-Control") == "no-cache" {
		// Log cache miss
		log.Printf("Request for %s - %s - %s missed cache", sanitizedUrl, r.RemoteAddr, r.Header.Get("Referer"))
		w.Header().Set("X-Cache", "MISS")

		// Send request
		imageFromUpstream, err := client.Get(serverResponse.ImageServer + sanitizedUrl)
		if err != nil {
			log.Printf("Request for %s failed: %v", serverResponse.ImageServer + sanitizedUrl, err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		defer imageFromUpstream.Body.Close()

		// Set Content-Length
		w.Header().Set("Content-Length", imageFromUpstream.Header.Get("Content-Length"))

		// Set timing header
		processedTime := time.Now().Sub(startTime).Milliseconds()
		w.Header().Set("X-Time-Taken", strconv.Itoa(int(processedTime)))
		log.Printf("Request for %s - %s - %s processed in %dms", sanitizedUrl, r.RemoteAddr, r.Header.Get("Referer"), processedTime)

		// Copy request to response body
		var imageBuffer bytes.Buffer
		io.Copy(w, io.TeeReader(imageFromUpstream.Body, &imageBuffer))

		// Save hash
		cache.Set(sanitizedUrl, imageBuffer.Bytes())
	} else {
		// Get length
		length := len(imageFromCache)
		image := make([]byte, length)
		copy(image, imageFromCache)

		// Log cache hit
		log.Printf("Request for %s - %s - %s hit cache", sanitizedUrl, r.RemoteAddr, r.Header.Get("Referer"))
		w.Header().Set("X-Cache", "HIT")

		// Set Content-Length
		w.Header().Set("Content-Length", strconv.Itoa(length))

		// Set timing header
		processedTime := time.Now().Sub(startTime).Milliseconds()
		w.Header().Set("X-Time-Taken", strconv.Itoa(int(processedTime)))
		log.Printf("Request for %s - %s - %s processed in %dms", sanitizedUrl, r.RemoteAddr, r.Header.Get("Referer"), processedTime)

		// Convert bytes object into reader and send to client
		imageReader := bytes.NewReader(image)
		io.Copy(w, imageReader)
	}

	// End time
	totalTime := time.Now().Sub(startTime).Milliseconds()
	w.Header().Set("X-Time-Taken", strconv.Itoa(int(totalTime)))
	log.Printf("Request for %s - %s - %s completed in %dms", sanitizedUrl, r.RemoteAddr, r.Header.Get("Referer"), totalTime)
}

func ShutdownHandler() {
	// Hook on to SIGTERM
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start coroutine to wait for SIGTERM
	go func() {
		<-c
		// Prepare to shutdown server
		fmt.Println("Shutting down server gracefully!")

		// Flip switch
		running = false

		// Sent stop request to backend
		request := ServerRequest{
			Secret: clientSettings.ClientSecret,
		}
		requestJson, _ := json.Marshal(&request)
		r, err := http.Post(apiBackend+"/stop", "application/json", bytes.NewBuffer(requestJson))
		if err != nil {
			log.Fatalf("Failed to shutdown server gracefully: %v", err)
		}
		defer r.Body.Close()

		// Wait till last request is normalised
		timeShutdown := time.Now()
		secondsSinceLastRequest := time.Now().Sub(timeLastRequest).Seconds()
		for secondsSinceLastRequest < 30 {
			log.Printf("%.2f seconds have elapsed since CTRL-C", secondsSinceLastRequest)

			// Give up after one minute
			if time.Now().Sub(timeShutdown).Seconds() > float64(clientSettings.GracefulShutdownInSeconds) {
				log.Printf("Giving up, quitting now!")
				break
			}

			// Count time :)
			time.Sleep(1 * time.Second)
			secondsSinceLastRequest = time.Now().Sub(timeLastRequest).Seconds()
		}

		// Exit properly
		os.Exit(0)
	}()
}

func main() {
	// Prepare logger
	logWriter := GetLogWriter()
	defer logWriter.Close()

	// Load client settings
	loadClientSettings()

	// Save client settings (in the event of new/invalid fields)
	saveClientSettings()

	// Create cache
	cache = diskcache.New(
		clientSettings.CacheDirectory,
		clientSettings.MaxCacheSizeInMebibytes*1024*1024,
		clientSettings.CacheScanIntervalInSeconds,
		clientSettings.MaxCacheScanTimeInSeconds,
	)
	defer cache.Close()

	// Prepare handlers
	r := mux.NewRouter()
	r.HandleFunc("/{image_type}/{chapter_hash}/{image_filename}", RequestHandler)
	r.HandleFunc("/{token}/{image_type}/{chapter_hash}/{image_filename}", RequestHandler)

	// Prepare server
	http.Handle("/", r)

	// Prepare client from retryablehttp
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	retryClient.Logger = nil

	// Override default tranport to allow for keep-alives
	transport := retryClient.HTTPClient.Transport.(*http.Transport)
	retryClient.HTTPClient.Transport = overrideKeepAliveHttpTransport(transport)

	// Create standard HTTP client from retryablehttp client
	client = retryClient.StandardClient()
	client.Timeout = time.Second * 15

	// Register shutdown handler
	ShutdownHandler()

	// Prepare certificates
	serverResponse = *pingServer()
	if serverResponse.Tls.Certificate == "" {
		log.Fatalln("Unable to contact API server!")
	}

	// Attempt to parse TLS data
	keyPair, err := tls.X509KeyPair([]byte(serverResponse.Tls.Certificate), []byte(serverResponse.Tls.PrivateKey))
	if err != nil {
		log.Fatalf("Cannot parse TLS data %v - %v", serverResponse, err)
	}

	// Start ping loop
	go BackgroundLoop()

	// Start proxy server
	err = ListenAndServeTLSKeyPair(":"+strconv.Itoa(clientSettings.ClientPort), keyPair, r)
	if err != nil {
		log.Fatalf("Cannot start server: %v", err)
	}
}
