package io

// SecurityConfig controls which I/O operations are permitted.
// Two-layer security: program must import the module AND runtime must enable it.
type SecurityConfig struct {
	EnabledModules []string           `json:"enabled_modules"`
	HTTP           HTTPSecurityConfig `json:"http"`
	FS             FSSecurityConfig   `json:"fs"`
	SQL            SQLSecurityConfig  `json:"sql"`
	Exec           ExecSecurityConfig `json:"exec"`
}

// HTTPSecurityConfig controls HTTP module security.
type HTTPSecurityConfig struct {
	AllowedHosts    []string `json:"allowed_hosts"`
	BlockedHosts    []string `json:"blocked_hosts"`
	MaxResponseSize int64    `json:"max_response_size"`
	Timeout         int      `json:"timeout"`
}

// FSSecurityConfig controls file system module security.
type FSSecurityConfig struct {
	AllowedPaths []string `json:"allowed_paths"`
	BlockedPaths []string `json:"blocked_paths"`
	MaxFileSize  int64    `json:"max_file_size"`
	AllowWrite   bool     `json:"allow_write"`
}

// SQLSecurityConfig controls SQL module security.
type SQLSecurityConfig struct {
	AllowedDrivers []string `json:"allowed_drivers"`
	MaxQueryTime   int      `json:"max_query_time"`
	MaxRows        int      `json:"max_rows"`
}

// ExecSecurityConfig controls command execution security.
type ExecSecurityConfig struct {
	AllowedCommands []string `json:"allowed_commands"`
	BlockedCommands []string `json:"blocked_commands"`
	MaxTimeout      int      `json:"max_timeout"`
	MaxOutputSize   int64    `json:"max_output_size"`
}
