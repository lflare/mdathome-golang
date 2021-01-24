package mdathome

// ClientSettings stores client settings
type ClientSettings struct {
	CacheDirectory            string `json:"cache_directory"`
	ClientPort                int    `json:"client_port"`
	OverridePortReport        int    `json:"override_port_report"`
	OverrideAddressReport     string `json:"override_address_report"`
	ClientSecret              string `json:"client_secret"`
	GracefulShutdownInSeconds int    `json:"graceful_shutdown_in_seconds"`

	MaxKilobitsPerSecond       int `json:"max_kilobits_per_second"`
	MaxCacheSizeInMebibytes    int `json:"max_cache_size_in_mebibytes"`
	MaxReportedSizeInMebibytes int `json:"max_reported_size_in_mebibytes"`

	CacheScanIntervalInSeconds int `json:"cache_scan_interval_in_seconds"`
	CacheRefreshAgeInSeconds   int `json:"cache_refresh_age_in_seconds"`
	MaxCacheScanTimeInSeconds  int `json:"max_cache_scan_time_in_seconds"`

	AllowHTTP2              bool   `json:"allow_http2"`
	AllowUpstreamPooling    bool   `json:"allow_upstream_pooling"`
	AllowVisitorRefresh     bool   `json:"allow_visitor_refresh"`
	EnablePrometheusMetrics bool   `json:"enable_prometheus_metrics"`
	MaxMindLicenseKey       string `json:"maxmind_license_key"`
	OverrideUpstream        string `json:"override_upstream"`
	RejectInvalidTokens     bool   `json:"reject_invalid_tokens"`
	VerifyImageIntegrity    bool   `json:"verify_image_integrity"`

	LogLevel              string `json:"log_level"`
	MaxLogSizeInMebibytes int    `json:"max_log_size_in_mebibytes"`
	MaxLogBackups         int    `json:"max_log_backups"`
	MaxLogAgeInDays       int    `json:"max_log_age_in_days"`
}

// ServerRequest stores a single `secret` field for miscellanous operations
type ServerRequest struct {
	Secret string `json:"secret"`
}

// ServerSettings stores server settings
type ServerSettings struct {
	Secret       string  `json:"secret"`
	Port         int     `json:"port"`
	IPAddress    string  `json:"ip_address,omitempty"`
	DiskSpace    int     `json:"disk_space"`
	NetworkSpeed int     `json:"network_speed"`
	BuildVersion int     `json:"build_version"`
	TLSCreatedAt *string `json:"tls_created_at"`
}

// TLSCert stores a representation of the TLS certificate issued by the API server
type TLSCert struct {
	CreatedAt   string `json:"created_at"`
	PrivateKey  string `json:"private_key"`
	Certificate string `json:"certificate"`
}

// ServerResponse stores a representation of the response given by the `/ping` backend
type ServerResponse struct {
	ImageServer string  `json:"image_server"`
	URL         string  `json:"url"`
	TokenKey    string  `json:"token_key"`
	Paused      bool    `json:"paused"`
	Compromised bool    `json:"compromised"`
	LatestBuild int     `json:"latest_build"`
	TLS         TLSCert `json:"tls"`
}

// Token stores a representation of a token hash issued by the backend
type Token struct {
	Expires string `json:"expires"`
	Hash    string `json:"hash"`
}
