package mdathome

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var controlClient *http.Client

func init() {
	// Prepare control server HTTP client
	controlClient = &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("tcp4", addr)
			},
		},
	}
}

func controlPing() *ServerResponse {
	// Prepare logger
	log := log.WithFields(logrus.Fields{"type": "control"})

	// Create settings JSON
	settings := ServerSettings{
		Secret:       viper.GetString("client.secret"),
		Port:         viper.GetInt("client.port"),
		DiskSpace:    viper.GetInt("cache.max_size_mebibytes") * 1024 * 1024, // 1GB
		NetworkSpeed: viper.GetInt("client.max_speed_kbps") * 1000 / 8,       // 100Mbps
		BuildVersion: ClientSpecification,
		TLSCreatedAt: nil,
	}

	// Override necessary settings
	if viper.GetInt("override.port") != 0 {
		settings.Port = viper.GetInt("override.port")
	}
	if viper.GetString("override.address") != "" {
		settings.IPAddress = viper.GetString("override.address")
	}
	if viper.GetInt("override.size") != 0 {
		settings.DiskSpace = viper.GetInt("override.size") * 1024 * 1024
	}

	// Marshal server settings to JSON
	settingsJSON, _ := json.Marshal(&settings)

	// Ping control server
	res, err := controlClient.Post(viper.GetString("client.control_server")+"/ping", "application/json", bytes.NewBuffer(settingsJSON))
	if err != nil {
		log.Errorf("Failed to ping control server: %v", err)
		return nil
	}
	defer res.Body.Close()

	// Read server response fully
	controlResponse, err := io.ReadAll(res.Body)
	if err != nil {
		log.Errorf("Failed to ping control server: %v", err)
		return nil
	}

	// Verify TLS certificate exists in response before proceeding
	tlsIndex := strings.Index(string(controlResponse), "\"tls\"")
	if tlsIndex == -1 {
		log.Errorf("Received invalid server response: %s", controlResponse)

		// If existing TLS certificate not already running in client, fail spectacularly
		if serverResponse.TLS.Certificate == "" {
			log.Fatalln("No valid TLS certificate found in memory, cannot continue!")
		}

		// Return early if unable to proceed
		return nil
	}
	log.Infof("Server settings received! - %s...", string(controlResponse[:tlsIndex]))

	// Decode & unmarshal server response
	newServerResponse := ServerResponse{}
	if err := json.Unmarshal(controlResponse, &newServerResponse); err != nil {
		log.Errorf("Failed to ping control server: %v", err)
		return nil
	}

	// Check response for valid image server
	if newServerResponse.ImageServer == "" {
		log.Printf("Failed to verify server response: %s", controlResponse)
		return nil
	}

	// Update client hostname in-memory
	clientURL, _ := url.Parse(newServerResponse.URL)
	clientHostname = clientURL.Hostname()

	// Return server response
	return &newServerResponse
}

func controlShutdown() {
	// Send stop request to control server
	request := ServerRequest{
		Secret: viper.GetString("client.secret"),
	}
	requestJSON, _ := json.Marshal(&request)
	if res, err := http.Post(viper.GetString("client.control_server")+"/stop", "application/json", bytes.NewBuffer(requestJSON)); err != nil {
		log.Fatalf("Failed to shutdown server gracefully: %v", err)
	} else {
		res.Body.Close()
	}
}

func controlGetCertificate() tls.Certificate {
	// Make control ping
	serverResponse = *controlPing()
	if serverResponse.TLS.Certificate == "" {
		log.Fatalln("Unable to contact API server!")
	}

	// Parse TLS certificate
	keyPair, err := tls.X509KeyPair([]byte(serverResponse.TLS.Certificate), []byte(serverResponse.TLS.PrivateKey))
	if err != nil {
		log.Fatalf("Cannot parse TLS data %v - %v", serverResponse, err)
	}

	// Return keyPair
	return keyPair
}
