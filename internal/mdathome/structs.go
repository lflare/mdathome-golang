package mdathome

// ClientSettings stores client settings
type ClientSettings struct {
	// Settings Versioning
	Version int `json:"version"`

	// Client
	LogDirectory              string `json:"log_directory"`
	CacheDirectory            string `json:"cache_directory"`
	GracefulShutdownInSeconds int    `json:"graceful_shutdown_in_seconds"`

	// Overrides
	OverridePortReport    int    `json:"override_port_report"`
	OverrideAddressReport string `json:"override_address_report"`
	OverrideSizeReport    int    `json:"override_size_report"`
	OverrideUpstream      string `json:"override_upstream"`

	// Node
	ClientPort              int    `json:"client_port"`
	ClientSecret            string `json:"client_secret"`
	MaxKilobitsPerSecond    int    `json:"max_kilobits_per_second"`
	MaxCacheSizeInMebibytes int    `json:"max_cache_size_in_mebibytes"`

	// Cache
	CacheScanIntervalInSeconds int `json:"cache_scan_interval_in_seconds"`
	CacheRefreshAgeInSeconds   int `json:"cache_refresh_age_in_seconds"`
	MaxCacheScanTimeInSeconds  int `json:"max_cache_scan_time_in_seconds"`

	// Performance
	AllowHTTP2           bool `json:"allow_http2"`
	AllowUpstreamPooling bool `json:"allow_upstream_pooling"`
	LowMemoryMode        bool `json:"low_memory_mode"`

	// Security
	AllowVisitorRefresh    bool `json:"allow_visitor_refresh"`
	RejectInvalidSNI       bool `json:"reject_invalid_sni"`
	RejectInvalidTokens    bool `json:"reject_invalid_tokens"`
	SendServerHeader       bool `json:"send_server_header"`
	UseReverseProxyHeaders bool `json:"use_reverse_proxy_ip"`
	VerifyImageIntegrity   bool `json:"verify_image_integrity"`

	// Metrics
	EnablePrometheusMetrics bool   `json:"enable_prometheus_metrics"`
	MaxMindLicenseKey       string `json:"maxmind_license_key"`

	// Log
	LogLevel              string `json:"log_level"`
	MaxLogSizeInMebibytes int    `json:"max_log_size_in_mebibytes"`
	MaxLogBackups         int    `json:"max_log_backups"`
	MaxLogAgeInDays       int    `json:"max_log_age_in_days"`

	// Development settings
	APIBackend string `json:"api_backend"`

	// Deprecated settings
	MaxReportedSizeInMebibytes int `json:"max_reported_size_in_mebibytes,omitempty"`
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
	ImageServer   string  `json:"image_server"`
	LatestBuild   int     `json:"latest_build"`
	URL           string  `json:"url"`
	TokenKey      string  `json:"token_key"`
	Compromised   bool    `json:"compromised"`
	Paused        bool    `json:"paused"`
	DisableTokens bool    `json:"disable_tokens"`
	TLS           TLSCert `json:"tls"`
}

// Token stores a representation of a token hash issued by the backend
type Token struct {
	Expires string `json:"expires"`
	Hash    string `json:"hash"`
}
