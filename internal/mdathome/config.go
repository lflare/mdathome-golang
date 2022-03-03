package mdathome

import (
	"os"

	"github.com/spf13/viper"
)

var defaultConfiguration string = `version = 2

[client]
control_server = "https://api.mangadex.network"
graceful_shutdown_seconds = 300
max_speed_kbps = 10000
port = 443
secret = ""

[override]
address = ""
port = 0
size = 0
upstream = ""

[cache]
directory = "cache/"
max_scan_interval_seconds = 300
max_scan_time_seconds = 60
max_size_mebibytes = 10240
refresh_age_seconds = 86400

[performance]
allow_http2 = true
client_timeout_seconds = 60
low_memory_mode = true
upstream_connection_reuse = true

[security]
allow_visitor_cache_refresh = false
reject_invalid_hostname = false
reject_invalid_sni = false
reject_invalid_tokens = true
send_server_header = false
use_forwarded_for_headers = false
verify_image_integrity = false

[metrics]
enable_prometheus = false
maxmind_license_key = ""

[log]
directory = "log/"
level = "info"
max_age_days = 7
max_backups = 3
max_size_mebibytes = 64
`

func prepareConfiguration() {
	// Configure Viper
	viper.AddConfigPath(".")
	viper.SetConfigName("config.toml")
	viper.SetConfigType("toml")

	// Load in configuration
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			log.Info("Configuration not found, creating!")
			if err := os.WriteFile("config.toml", []byte(defaultConfiguration), 0600); err != nil {
				log.Fatalf("Failed to write default configuration to 'config.toml'!")
			} else {
				log.Fatalf("Default configuration written to 'config.toml', please modify before running client again!")
			}
		} else {
			// Config file was found but another error was produced
			log.Errorf("Failed to read configuration: %v", err)
		}
	}

	// Configure auto-reload
	prepareConfigurationReload()
}
