package mdathome

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tcnksm/go-latest"
)

func saveClientSettings() {
	clientSettingsSampleBytes, err := json.MarshalIndent(clientSettings, "", "    ")
	if err != nil {
		log.Fatalln("Failed to marshal sample settings.json")
	}

	err = ioutil.WriteFile(ConfigFilePath, clientSettingsSampleBytes, 0600)
	if err != nil {
		log.Fatalf("Failed to create sample settings.json: %v", err)
	}
}

func loadClientSettings() {
	// Read JSON from file
	clientSettingsJSON, err := ioutil.ReadFile(ConfigFilePath)
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

	// Migrate settings to the latest version
	migrateClientSettings(&clientSettings)

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

func migrateClientSettings(cs *ClientSettings) {
	// Return early if fully migrated
	if cs.Version == ClientSettingsVersion {
		return
	}

	// Migrate from settings before version 1
	if cs.Version == 0 {
		cs.OverrideSizeReport = cs.MaxReportedSizeInMebibytes
		cs.MaxReportedSizeInMebibytes = 0
		cs.Version = 1
	}
}

func checkClientVersion() {
	// Prepare version check
	githubTag := &latest.GithubTag{
		Owner:             "lflare",
		Repository:        "mdathome-golang",
		FixVersionStrFunc: latest.DeleteFrontV(),
	}

	// Check if client is latest
	res, err := latest.Check(githubTag, ClientVersion)
	if err != nil {
		log.Printf("Failed to check client version %s? Proceed with caution!", ClientVersion)
	} else {
		if res.Outdated {
			log.Printf("Client %s is not the latest! You should update to the latest version %s now!", ClientVersion, res.Current)
			log.Printf("Client starting in 5 seconds...")
			time.Sleep(5 * time.Second)
		} else {
			log.Printf("Client %s is latest! Starting client!", ClientVersion)
		}
	}
}

func startBackgroundWorker() {
	// Wait 10 seconds
	log.Println("Starting background jobs!")
	time.Sleep(10 * time.Second)

	for running {
		// Reload client configuration
		log.Println("Reloading client configuration")
		loadClientSettings()

		// Update log level if need be
		newLogLevel, err := logrus.ParseLevel(clientSettings.LogLevel)
		if err == nil {
			log.SetLevel(newLogLevel)
		}

		// Update max cache size
		cache.UpdateCacheLimit(clientSettings.MaxCacheSizeInMebibytes * 1024 * 1024)
		cache.UpdateCacheScanInterval(clientSettings.CacheScanIntervalInSeconds)
		cache.UpdateCacheRefreshAge(clientSettings.CacheRefreshAgeInSeconds)

		// Update server response in a goroutine
		newServerResponse := backendPing()
		if newServerResponse != nil {
			// Check if overriding upstream
			if clientSettings.OverrideUpstream != "" {
				newServerResponse.ImageServer = clientSettings.OverrideUpstream
			}

			serverResponse = *newServerResponse
		}

		// Wait 10 seconds
		time.Sleep(10 * time.Second)
	}
}

func registerShutdownHandler() {
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

		// Send shutdown command to backend
		backendShutdown()

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
