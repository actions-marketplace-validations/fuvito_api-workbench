package domain

// Settings holds user preferences persisted across sessions.
type Settings struct {
	Theme            string `json:"theme"`            // "dark" | "light"
	FontSize         int    `json:"fontSize"`          // monospace font size in px (10–20)
	DefaultTimeoutMs int    `json:"defaultTimeoutMs"`  // per-request timeout; 0 = 30s
	ProxyURL         string `json:"proxyUrl"`          // http/https proxy; empty = none
	TLSSkipVerify    bool   `json:"tlsSkipVerify"`     // skip TLS certificate verification
}

// DefaultSettings returns the out-of-the-box defaults.
func DefaultSettings() Settings {
	return Settings{
		Theme:            "dark",
		FontSize:         13,
		DefaultTimeoutMs: 30000,
	}
}
