package mdathome

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

// Server ping handler
func backendPing() *ServerResponse {
	// Create settings JSON
	settings := ServerSettings{
		Secret:       clientSettings.ClientSecret,
		Port:         clientSettings.ClientPort,
		DiskSpace:    clientSettings.MaxReportedSizeInMebibytes * 1024 * 1024, // 1GB
		NetworkSpeed: clientSettings.MaxKilobitsPerSecond * 1000 / 8,          // 100Mbps
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

	// Marshal JSON
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

func backendShutdown() {
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
}
