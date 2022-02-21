package mdathome

// ServerRequest stores a single `secret` field for miscellaneous operations
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
