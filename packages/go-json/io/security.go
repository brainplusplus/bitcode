package io

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"
)

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

// DefaultSecurityConfig returns a restrictive default security configuration.
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		EnabledModules: nil,
		HTTP: HTTPSecurityConfig{
			BlockedHosts:    defaultBlockedHosts,
			MaxResponseSize: 10 * 1024 * 1024, // 10MB
			Timeout:         30,
		},
		FS: FSSecurityConfig{
			BlockedPaths: defaultBlockedPaths,
			MaxFileSize:  10 * 1024 * 1024, // 10MB
			AllowWrite:   false,
		},
		SQL: SQLSecurityConfig{
			MaxQueryTime: 30,
			MaxRows:      10000,
		},
		Exec: ExecSecurityConfig{
			MaxTimeout:    60,
			MaxOutputSize: 1024 * 1024, // 1MB
		},
	}
}

// EngineSecrets are environment variable names that are always stripped
// from inherited environments in exec operations.
var EngineSecrets = []string{
	"JWT_SECRET",
	"DB_PASSWORD",
	"ENCRYPTION_KEY",
	"SMTP_PASSWORD",
	"STORAGE_S3_SECRET_KEY",
	"STORAGE_S3_ACCESS_KEY",
}

// DeniedCommands are permanently blocked regardless of whitelist configuration.
var DeniedCommands = []string{
	"rm", "rmdir", "del", "format",
	"shutdown", "reboot", "halt", "poweroff",
	"dd", "mkfs", "fdisk",
}

var defaultBlockedHosts = []string{
	"localhost",
	"127.0.0.1",
	"::1",
	"0.0.0.0",
	"169.254.169.254",
}

var defaultBlockedPaths = []string{
	"/etc/",
	"/root/",
	"/proc/",
	"/sys/",
	"/dev/",
}

// localhostVariants maps all known localhost representations for blocking.
var localhostVariants = map[string]bool{
	"localhost": true,
	"127.0.0.1": true,
	"::1":       true,
	"0.0.0.0":   true,
	"[::1]":     true,
}

// ValidateHTTPRequest checks if a URL is allowed by the security config.
func (sc *SecurityConfig) ValidateHTTPRequest(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("security: invalid URL: %s", err.Error())
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("security: URL has no hostname")
	}

	// Check localhost variants (always blocked by default).
	if isLocalhostVariant(hostname) {
		if !isHostExplicitlyAllowed(hostname, sc.HTTP.AllowedHosts) {
			return fmt.Errorf("security: host '%s' is blocked (localhost)", hostname)
		}
	}

	// Check blocked hosts.
	for _, blocked := range sc.HTTP.BlockedHosts {
		if matchHost(hostname, blocked) {
			return fmt.Errorf("security: host '%s' is blocked", hostname)
		}
	}

	// Check cloud metadata endpoint.
	if isCloudMetadataIP(hostname) {
		return fmt.Errorf("security: host '%s' is blocked (cloud metadata endpoint)", hostname)
	}

	// If AllowedHosts is set, host must be in the list.
	if len(sc.HTTP.AllowedHosts) > 0 {
		allowed := false
		for _, ah := range sc.HTTP.AllowedHosts {
			if matchHost(hostname, ah) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("security: host '%s' is not in allowed hosts list", hostname)
		}
	}

	return nil
}

// ValidateFilePath checks if a file path is allowed by the security config.
func (sc *SecurityConfig) ValidateFilePath(path string, write bool) error {
	if write && !sc.FS.AllowWrite {
		return fmt.Errorf("security: write operations disabled by security config")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("security: cannot resolve path: %s", err.Error())
	}

	// Normalize to forward slashes for consistent comparison.
	absPath = filepath.ToSlash(absPath)

	// Check blocked paths.
	for _, blocked := range sc.FS.BlockedPaths {
		blocked = filepath.ToSlash(blocked)
		if strings.HasPrefix(absPath, blocked) || strings.HasPrefix(absPath+"/", blocked) {
			return fmt.Errorf("security: path '%s' is blocked", path)
		}
	}

	// If AllowedPaths is set, path must be under one of them.
	if len(sc.FS.AllowedPaths) > 0 {
		allowed := false
		for _, ap := range sc.FS.AllowedPaths {
			ap = filepath.ToSlash(ap)
			if strings.HasPrefix(absPath, ap) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("security: path '%s' is not in allowed paths", path)
		}
	}

	return nil
}

// ValidateCommand checks if a command is allowed by the security config.
func (sc *SecurityConfig) ValidateCommand(cmd string) error {
	cmdBase := filepath.Base(cmd)

	// DeniedCommands checked BEFORE whitelist — always blocked.
	for _, denied := range DeniedCommands {
		if strings.EqualFold(cmdBase, denied) {
			return fmt.Errorf("security: command '%s' is permanently blocked", cmd)
		}
	}

	// Check user-configured blocked commands.
	for _, blocked := range sc.Exec.BlockedCommands {
		if strings.EqualFold(cmdBase, blocked) {
			return fmt.Errorf("security: command '%s' is blocked", cmd)
		}
	}

	// Command must be in whitelist.
	if len(sc.Exec.AllowedCommands) == 0 {
		return fmt.Errorf("security: no commands are whitelisted — set AllowedCommands to allow execution")
	}

	for _, allowed := range sc.Exec.AllowedCommands {
		if strings.EqualFold(cmdBase, allowed) {
			return nil
		}
	}

	return fmt.Errorf("security: command '%s' not in allowed list", cmd)
}

// ValidateSQLDriver checks if a database driver is allowed by the security config.
func (sc *SecurityConfig) ValidateSQLDriver(driver string) error {
	if len(sc.SQL.AllowedDrivers) == 0 {
		return nil
	}

	for _, allowed := range sc.SQL.AllowedDrivers {
		if strings.EqualFold(driver, allowed) {
			return nil
		}
	}

	return fmt.Errorf("security: SQL driver '%s' not in allowed list", driver)
}

// IsModuleEnabled checks if a module name is in the enabled list.
func (sc *SecurityConfig) IsModuleEnabled(name string) bool {
	if sc == nil || len(sc.EnabledModules) == 0 {
		return false
	}
	for _, m := range sc.EnabledModules {
		if m == name {
			return true
		}
	}
	return false
}

// StripEngineSecrets removes sensitive environment variables from a map.
func StripEngineSecrets(env map[string]string) map[string]string {
	result := make(map[string]string, len(env))
	for k, v := range env {
		if isEngineSecret(k) {
			continue
		}
		result[k] = v
	}
	return result
}

func isEngineSecret(key string) bool {
	for _, secret := range EngineSecrets {
		if strings.EqualFold(key, secret) {
			return true
		}
	}
	return false
}

func isLocalhostVariant(hostname string) bool {
	if localhostVariants[hostname] {
		return true
	}
	// Check if it's an IP that resolves to loopback.
	ip := net.ParseIP(hostname)
	if ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

func isHostExplicitlyAllowed(hostname string, allowedHosts []string) bool {
	for _, ah := range allowedHosts {
		if matchHost(hostname, ah) {
			return true
		}
	}
	return false
}

func isCloudMetadataIP(hostname string) bool {
	// AWS/GCP/Azure metadata endpoint.
	if hostname == "169.254.169.254" {
		return true
	}
	// Azure Instance Metadata Service.
	if hostname == "169.254.169.253" {
		return true
	}
	// Link-local range used by cloud metadata.
	ip := net.ParseIP(hostname)
	if ip != nil {
		_, linkLocal, _ := net.ParseCIDR("169.254.0.0/16")
		if linkLocal != nil && linkLocal.Contains(ip) {
			return true
		}
	}
	return false
}

// matchHost checks if a hostname matches a pattern.
// Supports wildcard prefix: "*.example.com" matches "api.example.com".
func matchHost(hostname, pattern string) bool {
	if hostname == pattern {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // ".example.com"
		return strings.HasSuffix(hostname, suffix)
	}
	return false
}
