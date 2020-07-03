package main

type ClientSettings struct {
	CacheDirectory             string `json:"cache_directory"`
	ClientSecret               string `json:"client_secret"`
	ClientPort                 int    `json:"client_port"`
	MaxKilobitsPerSecond       int    `json:"max_kilobits_per_second"`
	MaxCacheSizeInMebibytes    int    `json:"max_cache_size_in_mebibytes"`
	MaxReportedSizeInMebibytes int    `json:"max_reported_size_in_mebibytes"`
	GracefulShutdownInSeconds  int    `json:"graceful_shutdown_in_seconds"`
	CacheScanIntervalInSeconds int    `json:"cache_scan_interval_in_seconds"`
	MaxCacheScanTimeInSeconds  int    `json:"max_cache_scan_time_in_seconds"`
}

type ServerRequest struct {
	Secret string `json:"secret"`
}

type ServerSettings struct {
	Secret       string  `json:"secret"`
	Port         int     `json:"port"`
	DiskSpace    int     `json:"disk_space"`
	NetworkSpeed int     `json:"network_speed"`
	BuildVersion int     `json:"build_version"`
	TlsCreatedAt *string `json:"tls_created_at"`
}

type TlsCert struct {
	CreatedAt   string `json:"created_at"`
	PrivateKey  string `json:"private_key"`
	Certificate string `json:"certificate"`
}

type ServerResponse struct {
	ImageServer string  `json:"image_server"`
	Url         string  `json:"url"`
	Paused      bool    `json:"paused"`
	Compromised bool    `json:"compromised"`
	LatestBuild int     `json:"latest_build"`
	Tls         TlsCert `json:"tls"`
}
