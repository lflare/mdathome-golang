package mdathome

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/tcnksm/go-latest"
)

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
	// Wait 15 seconds
	log.Println("Starting background jobs!")
	time.Sleep(15 * time.Second)

	for running {
		// Update log level if need be
		newLogLevel, err := logrus.ParseLevel(viper.GetString("log.level"))
		if err == nil {
			log.SetLevel(newLogLevel)
		}

		// Update server response in a goroutine
		newServerResponse := controlPing()
		if newServerResponse != nil {
			// Check if overriding upstream
			if viper.GetString("override.upstream") != "" {
				newServerResponse.ImageServer = viper.GetString("override.upstream")
			}

			serverResponse = *newServerResponse
		}

		// Wait 15 seconds
		time.Sleep(15 * time.Second)
	}
}

func registerShutdownHandler() {
	// Hook on to SIGTERM
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start coroutine to wait for SIGTERM
	go func() {
		<-c
		// Prepare to shutdown server
		fmt.Println("Shutting down server gracefully!")

		// Flip switch
		running = false

		// Send shutdown command to backend
		controlShutdown()

		// Wait till last request is normalised
		timeShutdown := time.Now()
		secondsSinceLastRequest := time.Since(timeLastRequest).Seconds()
		for secondsSinceLastRequest < 30 {
			log.Printf("%.2f seconds have elapsed since CTRL-C", secondsSinceLastRequest)

			// Give up after one minute
			if time.Since(timeShutdown).Seconds() > float64(viper.GetFloat64("client.graceful_shutdown_seconds")) {
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

// ByteCountIEC returns a human-readable string describing the size of bytes in int
func ByteCountIEC(b int) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

// ByTimestamp is a sortable slice of KeyPair based off timestamp
type ByTimestamp []KeyPair

func (a ByTimestamp) Len() int           { return len(a) }
func (a ByTimestamp) Less(i, j int) bool { return a[i].Timestamp < a[j].Timestamp }
func (a ByTimestamp) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
