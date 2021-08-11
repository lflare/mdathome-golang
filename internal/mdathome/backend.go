package mdathome

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// Server ping handler
func backendPing() *ServerResponse {
	// Create settings JSON
	settings := ServerSettings{
		Secret:       clientSettings.ClientSecret,
		Port:         clientSettings.ClientPort,
		DiskSpace:    clientSettings.MaxCacheSizeInMebibytes * 1024 * 1024, // 1GB
		NetworkSpeed: clientSettings.MaxKilobitsPerSecond * 1000 / 8,       // 100Mbps
		BuildVersion: ClientSpecification,
		TLSCreatedAt: nil,
	}

	// Check if we are overriding reported port
	if clientSettings.OverridePortReport != 0 {
		settings.Port = clientSettings.OverridePortReport
	}

	// Check if we are overriding reported address
	if clientSettings.OverrideAddressReport != "" {
		settings.IPAddress = clientSettings.OverrideAddressReport
	}

	// Check if we are overriding reported cache size
	if clientSettings.OverrideSizeReport != 0 {
		settings.DiskSpace = clientSettings.OverrideSizeReport * 1024 * 1024
	}

	// Marshal JSON
	settingsJSON, _ := json.Marshal(&settings)

	// Prepare backend client
	client = &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("tcp4", addr)
			},
		},
	}

	// Ping backend server
	r, err := client.Post(clientSettings.APIBackend+"/ping", "application/json", bytes.NewBuffer(settingsJSON))
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
	newServerResponse := ServerResponse{
		DisableTokens: false, // Default to not force disabling tokens
	}
	err = json.Unmarshal(response, &newServerResponse)
	if err != nil {
		log.Printf("Failed to ping control server: %v", err)
		return nil
	}

	// Check response for valid image server
	if newServerResponse.ImageServer == "" {
		log.Printf("Failed to verify server response: %s", response)
		return nil
	}

	// Update client hostname in-memory
	clientURL, _ := url.Parse(newServerResponse.URL)
	clientHostname = clientURL.Hostname()

	// Return server response
	return &newServerResponse
}

func backendShutdown() {
	// Sent stop request to backend
	request := ServerRequest{
		Secret: clientSettings.ClientSecret,
	}
	requestJSON, _ := json.Marshal(&request)
	r, err := http.Post(clientSettings.APIBackend+"/stop", "application/json", bytes.NewBuffer(requestJSON))
	if err != nil {
		log.Fatalf("Failed to shutdown server gracefully: %v", err)
	}
	defer r.Body.Close()
}

func backendGetCertificate() tls.Certificate {
	// Make backend ping
	serverResponse = *backendPing()
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
