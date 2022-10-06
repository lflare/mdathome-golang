package mdathome

import (
	"github.com/spf13/viper"
)

func setDefaultConfiguration() {
	// [version]
	viper.SetDefault("version", 2)

	// [client]
	viper.SetDefault("client.control_server", "https://api.mangadex.network")
	viper.SetDefault("client.graceful_shutdown_seconds", 300)
	viper.SetDefault("client.max_speed_kbps", 10000)
	viper.SetDefault("client.port", 443)
	viper.SetDefault("client.secret", "")

	// [override]
	viper.SetDefault("override.address", "")
	viper.SetDefault("override.port", 0)
	viper.SetDefault("override.size", 0)
	viper.SetDefault("override.upstream", "")

	// [cache]
	viper.SetDefault("cache.directory", "cache/")
	viper.SetDefault("cache.max_scan_interval_seconds", 900)
	viper.SetDefault("cache.max_scan_time_seconds", 300)
	viper.SetDefault("cache.max_size_mebibytes", 10240)
	viper.SetDefault("cache.refresh_age_seconds", 86400)

	// [performance]
	viper.SetDefault("performance.allow_http2", true)
	viper.SetDefault("performance.client_timeout_seconds", 60)
	viper.SetDefault("performance.low_memory_mode", true)
	viper.SetDefault("performance.upstream_connection_reuse", true)

	// [security]
	viper.SetDefault("security.allow_visitor_cache_refresh", false)
	viper.SetDefault("security.reject_invalid_hostname", false)
	viper.SetDefault("security.reject_invalid_sni", false)
	viper.SetDefault("security.reject_invalid_tokens", true)
	viper.SetDefault("security.send_server_header", false)
	viper.SetDefault("security.use_forwarded_for_headers", false)
	viper.SetDefault("security.verify_image_integrity", false)

	// [metric]
	viper.SetDefault("metrics.enable_prometheus", false)
	viper.SetDefault("metrics.enable_geoip", false)
	viper.SetDefault("metrics.maxmind_license_key", "")

	// [log]
	viper.SetDefault("log.directory", "log/")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.max_age_days", 7)
	viper.SetDefault("log.max_backups", 3)
	viper.SetDefault("log.max_size_mebibytes", 64)
}

func prepareConfiguration() {
	// Configure Viper
	configFile := "config.toml"
	configType := "toml"
	viper.AddConfigPath(".")
	viper.SetConfigFile(configFile)
	viper.SetConfigType(configType)

	// Set default configuration
	setDefaultConfiguration()

	// Load in configuration
	if err := viper.ReadInConfig(); err != nil {
		// Write default configuration file if not exists
		log.Infof("Could not read configuration: '%v', attempting to create configuration!", err)
		if err := viper.SafeWriteConfig(); err != nil {
			log.Fatalf("Failed to write default configuration to '%s': %v", configFile, err)
		} else {
			log.Fatalf("Default configuration written to '%s', please modify before running client again!", configFile)
		}
	}

	// Update default configuration file
	if err := viper.WriteConfig(); err != nil {
		log.Errorf("Failed to update configuration file: '%v'. Please check permissions!", err)
	}

	// Configure auto-reload
	prepareConfigurationReload()
}
