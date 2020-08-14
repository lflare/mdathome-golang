package main

// ClientSettings stores client settings
type ClientSettings struct {
	CacheDirectory             string `json:"cache_directory"`
	ClientSecret               string `json:"client_secret"`
	ClientPort                 int    `json:"client_port"`
	MaxKilobitsPerSecond       int    `json:"max_kilobits_per_second"`
	MaxCacheSizeInMebibytes    int    `json:"max_cache_size_in_mebibytes"`
	MaxReportedSizeInMebibytes int    `json:"max_reported_size_in_mebibytes"`
	GracefulShutdownInSeconds  int    `json:"graceful_shutdown_in_seconds"`
	CacheScanIntervalInSeconds int    `json:"cache_scan_interval_in_seconds"`
	CacheRefreshAgeInSeconds   int    `json:"cache_refresh_age_in_seconds"`
	MaxCacheScanTimeInSeconds  int    `json:"max_cache_scan_time_in_seconds"`
	RejectInvalidTokens        bool   `json:"reject_invalid_tokens"`
}

// ServerRequest stores a single `secret` field for miscellanous operations
type ServerRequest struct {
	Secret string `json:"secret"`
}

// ServerSettings stores server settings
type ServerSettings struct {
	Secret       string  `json:"secret"`
	Port         int     `json:"port"`
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
