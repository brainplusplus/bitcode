package io

import (
	"testing"
)

func TestValidateHTTPRequest_BlockedHosts(t *testing.T) {
	sc := DefaultSecurityConfig()

	tests := []struct {
		url     string
		blocked bool
	}{
		{"http://localhost:8080/api", true},
		{"http://127.0.0.1:3000/data", true},
		{"http://[::1]/api", true},
		{"http://0.0.0.0/api", true},
		{"http://169.254.169.254/metadata", true},
		{"http://169.254.169.253/metadata", true},
		{"https://api.example.com/users", false},
	}

	for _, tt := range tests {
		err := sc.ValidateHTTPRequest(tt.url)
		if tt.blocked && err == nil {
			t.Errorf("expected %s to be blocked", tt.url)
		}
		if !tt.blocked && err != nil {
			t.Errorf("expected %s to be allowed, got: %s", tt.url, err.Error())
		}
	}
}

func TestValidateHTTPRequest_AllowedHosts(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.HTTP.AllowedHosts = []string{"api.example.com", "*.internal.com"}

	tests := []struct {
		url     string
		allowed bool
	}{
		{"https://api.example.com/users", true},
		{"https://service.internal.com/data", true},
		{"https://other.com/api", false},
	}

	for _, tt := range tests {
		err := sc.ValidateHTTPRequest(tt.url)
		if tt.allowed && err != nil {
			t.Errorf("expected %s to be allowed, got: %s", tt.url, err.Error())
		}
		if !tt.allowed && err == nil {
			t.Errorf("expected %s to be blocked", tt.url)
		}
	}
}

func TestValidateHTTPRequest_WildcardHost(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.HTTP.AllowedHosts = []string{"*.internal.com"}

	err := sc.ValidateHTTPRequest("https://api.internal.com/data")
	if err != nil {
		t.Errorf("wildcard should match: %s", err.Error())
	}

	err = sc.ValidateHTTPRequest("https://internal.com/data")
	if err == nil {
		t.Error("wildcard *.internal.com should not match internal.com")
	}
}

func TestValidateFilePath_PathTraversal(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.FS.AllowedPaths = []string{"/tmp/go-json/"}

	err := sc.ValidateFilePath("../../etc/passwd", false)
	if err == nil {
		t.Error("path traversal should be blocked")
	}
}

func TestValidateFilePath_WriteDisabled(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.FS.AllowWrite = false

	err := sc.ValidateFilePath("/tmp/test.txt", true)
	if err == nil {
		t.Error("write should be blocked when AllowWrite is false")
	}
}

func TestValidateFilePath_WriteEnabled(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.FS.AllowWrite = true
	sc.FS.AllowedPaths = nil
	sc.FS.BlockedPaths = nil

	err := sc.ValidateFilePath("/tmp/test.txt", true)
	if err != nil {
		t.Errorf("write should be allowed: %s", err.Error())
	}
}

func TestValidateCommand_DeniedCommands(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.Exec.AllowedCommands = []string{"rm", "echo"}

	err := sc.ValidateCommand("rm")
	if err == nil {
		t.Error("rm should be permanently blocked even if in AllowedCommands")
	}
}

func TestValidateCommand_AllowedCommands(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.Exec.AllowedCommands = []string{"echo", "pandoc"}

	err := sc.ValidateCommand("echo")
	if err != nil {
		t.Errorf("echo should be allowed: %s", err.Error())
	}

	err = sc.ValidateCommand("curl")
	if err == nil {
		t.Error("curl should not be allowed")
	}
}

func TestValidateCommand_NoWhitelist(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.Exec.AllowedCommands = nil

	err := sc.ValidateCommand("echo")
	if err == nil {
		t.Error("should fail when no commands are whitelisted")
	}
}

func TestValidateSQLDriver(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.SQL.AllowedDrivers = []string{"sqlite3", "postgres"}

	err := sc.ValidateSQLDriver("sqlite3")
	if err != nil {
		t.Errorf("sqlite3 should be allowed: %s", err.Error())
	}

	err = sc.ValidateSQLDriver("mysql")
	if err == nil {
		t.Error("mysql should not be allowed")
	}
}

func TestValidateSQLDriver_NoRestriction(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.SQL.AllowedDrivers = nil

	err := sc.ValidateSQLDriver("anything")
	if err != nil {
		t.Errorf("should allow any driver when no restriction: %s", err.Error())
	}
}

func TestStripEngineSecrets(t *testing.T) {
	env := map[string]string{
		"PATH":                   "/usr/bin",
		"HOME":                   "/home/user",
		"JWT_SECRET":             "secret123",
		"DB_PASSWORD":            "pass123",
		"ENCRYPTION_KEY":         "key123",
		"SMTP_PASSWORD":          "smtp123",
		"STORAGE_S3_SECRET_KEY":  "s3secret",
		"STORAGE_S3_ACCESS_KEY":  "s3access",
		"CUSTOM_VAR":             "value",
	}

	result := StripEngineSecrets(env)

	if _, ok := result["JWT_SECRET"]; ok {
		t.Error("JWT_SECRET should be stripped")
	}
	if _, ok := result["DB_PASSWORD"]; ok {
		t.Error("DB_PASSWORD should be stripped")
	}
	if _, ok := result["PATH"]; !ok {
		t.Error("PATH should be preserved")
	}
	if _, ok := result["CUSTOM_VAR"]; !ok {
		t.Error("CUSTOM_VAR should be preserved")
	}
	if len(result) != 3 {
		t.Errorf("expected 3 remaining vars, got %d", len(result))
	}
}

func TestIsModuleEnabled(t *testing.T) {
	sc := &SecurityConfig{EnabledModules: []string{"http", "fs"}}

	if !sc.IsModuleEnabled("http") {
		t.Error("http should be enabled")
	}
	if !sc.IsModuleEnabled("fs") {
		t.Error("fs should be enabled")
	}
	if sc.IsModuleEnabled("sql") {
		t.Error("sql should not be enabled")
	}
}

func TestMatchHost(t *testing.T) {
	tests := []struct {
		hostname string
		pattern  string
		expected bool
	}{
		{"api.example.com", "api.example.com", true},
		{"api.example.com", "*.example.com", true},
		{"example.com", "*.example.com", false},
		{"other.com", "api.example.com", false},
	}

	for _, tt := range tests {
		result := matchHost(tt.hostname, tt.pattern)
		if result != tt.expected {
			t.Errorf("matchHost(%q, %q) = %v, want %v", tt.hostname, tt.pattern, result, tt.expected)
		}
	}
}
