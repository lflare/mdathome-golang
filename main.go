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
	"github.com/lflare/diskcache-golang"
	"github.com/tcnksm/go-latest"
	"golang.org/x/crypto/nacl/box"
)

const clientVersion string = "v1.3.2"
const specVersion int = 16

var clientSettings = ClientSettings{
	CacheDirectory:             "cache/", // Default cache directory
	ClientPort:                 44300,    // Default client port
	MaxKilobitsPerSecond:       10000,    // Default 10Mbps
	MaxCacheSizeInMebibytes:    10240,    // Default 10GB
	MaxReportedSizeInMebibytes: 10240,    // Default 10GB
	GracefulShutdownInSeconds:  300,      // Default 5m graceful shutdown
	CacheScanIntervalInSeconds: 300,      // Default 5m scan interval
	CacheRefreshAgeInSeconds:   3600,     // Default 1h cache refresh age
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
	clientSettingsJSON, err := ioutil.ReadFile("settings.json")
	if err != nil {
		log.Printf("Failed to read client configuration file - %v", err)
		saveClientSettings()
		log.Fatalf("Created sample settings.json! Please edit it before running again!")
	}

	// Unmarshal JSON to clientSettings struct
	err = json.Unmarshal(clientSettingsJSON, &clientSettings)
	if err != nil {
		log.Fatalf("Unable to unmarshal JSON file: %v", err)
	}

	// Check client configuration
	if clientSettings.ClientSecret == "" {
		log.Fatalf("Empty secret! Cannot run!")
	}

	if clientSettings.CacheDirectory == "" {
		log.Fatalf("Empty cache directory! Cannot run!")
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
		DiskSpace:    clientSettings.MaxReportedSizeInMebibytes * 1024 * 1024, // 1GB
		NetworkSpeed: clientSettings.MaxKilobitsPerSecond * 1000 / 8,          // 100Mbps
		BuildVersion: specVersion,
		TLSCreatedAt: nil,
	}
	settingsJSON, _ := json.Marshal(&settings)

	// Ping backend server
	r, err := http.Post(apiBackend+"/ping", "application/json", bytes.NewBuffer(settingsJSON))
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

		if serverResponse.TLS.Certificate == "" {
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
func backgroundLoop() {
	// Wait 15 seconds
	log.Println("Starting background jobs!")
	time.Sleep(15 * time.Second)

	for running {
		// Reload client configuration
		log.Println("Reloading client configuration")
		loadClientSettings()

		// Update max cache size
		cache.UpdateCacheLimit(clientSettings.MaxCacheSizeInMebibytes * 1024 * 1024)
		cache.UpdateCacheScanInterval(clientSettings.CacheScanIntervalInSeconds)
		cache.UpdateCacheRefreshAge(clientSettings.CacheRefreshAgeInSeconds)

		// Update server response in a goroutine
		newServerResponse := pingServer()
		if newServerResponse != nil {
			serverResponse = *newServerResponse
		}

		// Wait 15 seconds
		time.Sleep(15 * time.Second)
	}
}

func verifyToken(tokenString string, chapterHash string) (int, error) {
	// Check if given token string is empty
	if tokenString == "" {
		return 403, fmt.Errorf("Token is empty")
	}

	// Decode base64-encoded token & key
	tokenBytes, err := base64.RawURLEncoding.DecodeString(tokenString)
	if err != nil {
		return 403, fmt.Errorf("Cannot decode token - %v", err)
	}
	keyBytes, err := base64.StdEncoding.DecodeString(serverResponse.TokenKey)
	if err != nil {
		return 403, fmt.Errorf("Cannot decode key - %v", err)
	}

	// Copy over byte slices to fixed-length byte arrays for decryption
	var nonce [24]byte
	copy(nonce[:], tokenBytes[:24])
	var key [32]byte
	copy(key[:], keyBytes[:32])

	// Decrypt token
	data, ok := box.OpenAfterPrecomputation(nil, tokenBytes[24:], &nonce, &key)
	if !ok {
		return 403, fmt.Errorf("Failed to decrypt token")
	}

	// Unmarshal to struct
	token := Token{}
	if err := json.Unmarshal(data, &token); err != nil {
		return 403, fmt.Errorf("Failed to unmarshal token - %v", err)
	}

	// Parse expiry time
	expires, err := time.Parse(time.RFC3339, token.Expires)
	if err != nil {
		return 403, fmt.Errorf("Failed to parse expiry from token - %v", err)
	}

	// Check token expiry timing
	if time.Now().After(expires) {
		return 410, fmt.Errorf("Token expired")
	}

	// Check that chapter hashes are the same
	if token.Hash != chapterHash {
		return 403, fmt.Errorf("Token hash invalid")
	}

	// Token is valid
	return 0, nil
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	// Start timer
	startTime := time.Now()

	// Check if referer exists, else fake one
	if r.Header.Get("Referer") == "" {
		r.Header.Set("Referer", "None")
	}

	// Extract tokens
	tokens := mux.Vars(r)

	// Sanitized URL
	if tokens["image_type"] != "data" && tokens["image_type"] != "data-saver" {
		log.Printf("Request for %s - %s - %s failed", r.URL.Path, r.RemoteAddr, r.Header.Get("Referer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if matched, _ := regexp.MatchString(`^[0-9a-f]{32}$`, tokens["chapter_hash"]); !matched {
		log.Printf("Request for %s - %s - %s failed", r.URL.Path, r.RemoteAddr, r.Header.Get("Referer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if matched, _ := regexp.MatchString(`^.+\.(jpg|jpeg|png|gif)$`, tokens["image_filename"]); !matched {
		log.Printf("Request for %s - %s - %s failed", r.URL.Path, r.RemoteAddr, r.Header.Get("Referer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if token is valid
	code, err := verifyToken(tokens["token"], tokens["chapter_hash"])
	if err != nil {
		log.Printf("Request for %s - %s - %s rejected: %v", r.URL.Path, r.RemoteAddr, r.Header.Get("Referer"), err)

		// Reject invalid tokens if we enabled it
		if clientSettings.RejectInvalidTokens {
			w.WriteHeader(code)
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
	log.Printf("Request for %s - %s - %s received", sanitizedURL, r.RemoteAddr, r.Header.Get("Referer"))

	// Check if browser token exists
	if r.Header.Get("If-Modified-Since") != "" {
		// Log browser cache
		log.Printf("Request for %s - %s - %s cached by browser", sanitizedURL, r.RemoteAddr, r.Header.Get("Referer"))
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Load cache
	imageFromCache, err := cache.Get(sanitizedURL)

	// Check if image is correct type if in cache
	ok := (err == nil)
	if ok {
		contentType := http.DetectContentType(imageFromCache)
		if !strings.Contains(contentType, "image") {
			ok = false
		}
	}

	// Check if image exists and is a proper image and if cache-control is set
	if !ok || r.Header.Get("Cache-Control") == "no-cache" {
		// Log cache miss
		log.Printf("Request for %s - %s - %s missed cache", sanitizedURL, r.RemoteAddr, r.Header.Get("Referer"))
		w.Header().Set("X-Cache", "MISS")

		// Send request
		imageFromUpstream, err := client.Get(serverResponse.ImageServer + sanitizedURL)
		if err != nil {
			log.Printf("Request for %s failed: %v", serverResponse.ImageServer+sanitizedURL, err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		defer imageFromUpstream.Body.Close()

		// If not 200
		if imageFromUpstream.StatusCode != 200 {
			log.Printf("Request for %s failed: %d", serverResponse.ImageServer+sanitizedURL, imageFromUpstream.StatusCode)
			w.WriteHeader(imageFromUpstream.StatusCode)
			return
		}

		// Set Content-Length
		w.Header().Set("Content-Length", imageFromUpstream.Header.Get("Content-Length"))

		// Set timing header
		processedTime := time.Since(startTime).Milliseconds()
		w.Header().Set("X-Time-Taken", strconv.Itoa(int(processedTime)))
		log.Printf("Request for %s - %s - %s processed in %dms", sanitizedURL, r.RemoteAddr, r.Header.Get("Referer"), processedTime)

		// Copy request to response body
		var imageBuffer bytes.Buffer
		_, err = io.Copy(w, io.TeeReader(imageFromUpstream.Body, &imageBuffer))

		// Check if image was streamed properly
		if err != nil {
			log.Printf("Request for %s failed: %v", serverResponse.ImageServer+sanitizedURL, err)
			return
		}

		// Save hash
		err = cache.Set(sanitizedURL, imageBuffer.Bytes())
		if err != nil {
			log.Printf("Unexpected error encountered when saving image to cache: %v", err)
		}
	} else {
		// Get length
		length := len(imageFromCache)
		image := make([]byte, length)
		copy(image, imageFromCache)

		// Log cache hit
		log.Printf("Request for %s - %s - %s hit cache", sanitizedURL, r.RemoteAddr, r.Header.Get("Referer"))
		w.Header().Set("X-Cache", "HIT")

		// Set Content-Length
		w.Header().Set("Content-Length", strconv.Itoa(length))

		// Set timing header
		processedTime := time.Since(startTime).Milliseconds()
		w.Header().Set("X-Time-Taken", strconv.Itoa(int(processedTime)))
		log.Printf("Request for %s - %s - %s processed in %dms", sanitizedURL, r.RemoteAddr, r.Header.Get("Referer"), processedTime)

		// Convert bytes object into reader and send to client
		imageReader := bytes.NewReader(image)
		_, err := io.Copy(w, imageReader)

		// Check if image was streamed properly
		if err != nil {
			log.Printf("Request for %s failed: %v", serverResponse.ImageServer+sanitizedURL, err)
			return
		}
	}

	// End time
	totalTime := time.Since(startTime).Milliseconds()
	w.Header().Set("X-Time-Taken", strconv.Itoa(int(totalTime)))
	log.Printf("Request for %s - %s - %s completed in %dms", sanitizedURL, r.RemoteAddr, r.Header.Get("Referer"), totalTime)
}

func shutdownHandler() {
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
		requestJSON, _ := json.Marshal(&request)
		r, err := http.Post(apiBackend+"/stop", "application/json", bytes.NewBuffer(requestJSON))
		if err != nil {
			log.Fatalf("Failed to shutdown server gracefully: %v", err)
		}
		defer r.Body.Close()

		// Wait till last request is normalised
		timeShutdown := time.Now()
		secondsSinceLastRequest := time.Since(timeLastRequest).Seconds()
		for secondsSinceLastRequest < 30 {
			log.Printf("%.2f seconds have elapsed since CTRL-C", secondsSinceLastRequest)

			// Give up after one minute
			if time.Since(timeShutdown).Seconds() > float64(clientSettings.GracefulShutdownInSeconds) {
				log.Printf("Giving up, quitting now!")
				break
			}

			// Count time :)
			time.Sleep(1 * time.Second)
			secondsSinceLastRequest = time.Since(timeLastRequest).Seconds()
		}

		// Exit properly
		os.Exit(0)
	}()
}

func checkclientVersion() {
	// Prepare version check
	githubTag := &latest.GithubTag{
		Owner:             "lflare",
		Repository:        "mdathome-golang",
		FixVersionStrFunc: latest.DeleteFrontV(),
	}

	// Check if client is latest
	res, err := latest.Check(githubTag, clientVersion)
	if err != nil {
		log.Printf("Failed to check client version %s? Proceed with caution!", clientVersion)
	} else {
		if res.Outdated {
			log.Printf("Client %s is not the latest! You should update to the latest version %s now!", clientVersion, res.Current)
			log.Printf("Client starting in 10 seconds...")
			time.Sleep(10 * time.Second)
		} else {
			log.Printf("Client %s is latest! Starting client!", clientVersion)
		}
	}
}

func main() {
	// Prepare logger
	logWriter := getLogWriter()
	defer logWriter.Close()

	// Check client version
	checkclientVersion()

	// Load client settings
	loadClientSettings()

	// Save client settings (in the event of new/invalid fields)
	saveClientSettings()

	// Create cache
	cache = diskcache.New(
		clientSettings.CacheDirectory,
		clientSettings.MaxCacheSizeInMebibytes*1024*1024,
		clientSettings.CacheScanIntervalInSeconds,
		clientSettings.CacheRefreshAgeInSeconds,
		clientSettings.MaxCacheScanTimeInSeconds,
	)
	defer cache.Close()

	// Prepare handlers
	r := mux.NewRouter()
	r.HandleFunc("/{image_type}/{chapter_hash}/{image_filename}", requestHandler)
	r.HandleFunc("/{token}/{image_type}/{chapter_hash}/{image_filename}", requestHandler)

	// Prepare server
	http.Handle("/", r)

	// Create client
	tr := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 60 * time.Second,
	}
	client = &http.Client{Transport: tr}

	// Register shutdown handler
	shutdownHandler()

	// Prepare certificates
	serverResponse = *pingServer()
	if serverResponse.TLS.Certificate == "" {
		log.Fatalln("Unable to contact API server!")
	}

	// Attempt to parse TLS data
	keyPair, err := tls.X509KeyPair([]byte(serverResponse.TLS.Certificate), []byte(serverResponse.TLS.PrivateKey))
	if err != nil {
		log.Fatalf("Cannot parse TLS data %v - %v", serverResponse, err)
	}

	// Start ping loop
	go backgroundLoop()

	// Start proxy server
	err = listenAndServeTLSKeyPair(":"+strconv.Itoa(clientSettings.ClientPort), keyPair, r)
	if err != nil {
		log.Fatalf("Cannot start server: %v", err)
	}
}
