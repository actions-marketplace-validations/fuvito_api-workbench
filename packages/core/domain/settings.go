package domain

// Settings holds user preferences persisted across sessions.
type Settings struct {
	Theme            string `json:"theme"`            // "dark" | "light"
	FontSize         int    `json:"fontSize"`         // monospace font size in px (10–20)
	DefaultTimeoutMs int    `json:"defaultTimeoutMs"` // per-request timeout; 0 = 30s
	ProxyURL         string `json:"proxyUrl"`         // http/https proxy; empty = none
	TLSSkipVerify    bool   `json:"tlsSkipVerify"`    // skip TLS certificate verification
	TelemetryEnabled bool   `json:"telemetryEnabled"` // opt-in anonymous usage telemetry
	// Mutual TLS: paths only (a ${NAME} reference is expanded when a run starts); the key and
	// certificate contents are never stored in settings or project files.
	ClientCertFile string `json:"clientCertFile,omitempty"`
	ClientKeyFile  string `json:"clientKeyFile,omitempty"`
	CAFile         string `json:"caFile,omitempty"` // extra CA certificates to trust

	// Ports of the built-in Dev Servers. 0 = automatic: the first free port from the server's
	// usual starting port. A port that is set and already in use is an error, never silently
	// replaced.
	DemoServerPort        int `json:"demoServerPort,omitempty"`
	ConformanceServerPort int `json:"conformanceServerPort,omitempty"`
}

// DefaultSettings returns the out-of-the-box defaults.
func DefaultSettings() Settings {
	return Settings{
		Theme:            "dark",
		FontSize:         13,
		DefaultTimeoutMs: 30000,
	}
}
