package bridge

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestBridgeError(t *testing.T) {
	err := NewError(ErrRecordNotFound, "record not found in lead")
	if err.Code != ErrRecordNotFound {
		t.Errorf("expected code %s, got %s", ErrRecordNotFound, err.Code)
	}
	if err.Retryable {
		t.Error("expected non-retryable")
	}
	if err.Error() != "[RECORD_NOT_FOUND] record not found in lead" {
		t.Errorf("unexpected Error(): %s", err.Error())
	}
}

func TestBridgeErrorRetryable(t *testing.T) {
	err := NewRetryableError(ErrHTTPTimeout, "request timed out")
	if !err.Retryable {
		t.Error("expected retryable")
	}
}

func TestBridgeErrorWithDetails(t *testing.T) {
	err := ErrRecordNotFoundFor("lead", "abc-123")
	if err.Code != ErrRecordNotFound {
		t.Errorf("expected code %s, got %s", ErrRecordNotFound, err.Code)
	}
	if err.Details["model"] != "lead" {
		t.Errorf("expected model=lead, got %v", err.Details["model"])
	}
	if err.Details["id"] != "abc-123" {
		t.Errorf("expected id=abc-123, got %v", err.Details["id"])
	}
}

func TestBridgeErrorPermissionDenied(t *testing.T) {
	err := ErrPermissionDeniedFor("lead", "delete")
	if err.Code != ErrPermissionDenied {
		t.Errorf("expected code %s, got %s", ErrPermissionDenied, err.Code)
	}
	if err.Details["operation"] != "delete" {
		t.Errorf("expected operation=delete, got %v", err.Details["operation"])
	}
}

func TestBridgeErrorSudoNotAllowed(t *testing.T) {
	err := ErrSudoNotAllowedFor("crm")
	if err.Code != ErrSudoNotAllowed {
		t.Errorf("expected code %s, got %s", ErrSudoNotAllowed, err.Code)
	}
	if err.Details["module"] != "crm" {
		t.Errorf("expected module=crm, got %v", err.Details["module"])
	}
}

func TestEnvBridgeDenyEngineSecrets(t *testing.T) {
	v := viper.New()
	env := newEnvBridge(v, SecurityRules{EnvAllow: []string{"*"}}, "test")

	for _, secret := range EngineSecrets {
		_, err := env.Get(secret)
		if err == nil {
			t.Errorf("expected error for engine secret %s", secret)
		}
		if bridgeErr, ok := err.(*BridgeError); ok {
			if bridgeErr.Code != ErrEnvAccessDenied {
				t.Errorf("expected %s for %s, got %s", ErrEnvAccessDenied, secret, bridgeErr.Code)
			}
		}
	}
}

func TestEnvBridgeDenyList(t *testing.T) {
	v := viper.New()
	env := newEnvBridge(v, SecurityRules{
		EnvAllow: []string{"*"},
		EnvDeny:  []string{"STRIPE_*"},
	}, "test")

	_, err := env.Get("STRIPE_API_KEY")
	if err == nil {
		t.Error("expected error for denied env key STRIPE_API_KEY")
	}
}

func TestEnvBridgeModulePrefixOnly(t *testing.T) {
	v := viper.New()
	v.Set("CRM_API_KEY", "test-key")
	env := newEnvBridge(v, SecurityRules{}, "crm")

	_, err := env.Get("HRM_SECRET")
	if err == nil {
		t.Error("expected error for cross-module env key HRM_SECRET")
	}

	_, err = env.Get("CRM_API_KEY")
	if err != nil {
		t.Errorf("expected no error for own module env key CRM_API_KEY, got %v", err)
	}
}

func TestExecBridgeDenyGlobalCommands(t *testing.T) {
	exec := newExecBridge(SecurityRules{ExecAllow: []string{"*"}})

	for _, cmd := range DeniedCommands {
		_, err := exec.Exec(cmd, nil, nil)
		if err == nil {
			t.Errorf("expected error for denied command %s", cmd)
		}
	}
}

func TestExecBridgeNoAllowList(t *testing.T) {
	exec := newExecBridge(SecurityRules{})

	_, err := exec.Exec("pandoc", nil, nil)
	if err == nil {
		t.Error("expected error when exec_allow is empty")
	}
	if bridgeErr, ok := err.(*BridgeError); ok {
		if bridgeErr.Code != ErrExecDenied {
			t.Errorf("expected %s, got %s", ErrExecDenied, bridgeErr.Code)
		}
	}
}

func TestExecBridgeAllowList(t *testing.T) {
	exec := newExecBridge(SecurityRules{ExecAllow: []string{"echo"}})

	_, err := exec.Exec("curl", nil, nil)
	if err == nil {
		t.Error("expected error for command not in allow list")
	}
}

func TestExecBridgeDenyOverridesAllow(t *testing.T) {
	exec := newExecBridge(SecurityRules{
		ExecAllow: []string{"rm"},
		ExecDeny:  []string{"rm"},
	})

	_, err := exec.Exec("rm", nil, nil)
	if err == nil {
		t.Error("expected error: deny should override allow")
	}
}

func TestFSBridgeRelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	fs := newFSBridge(tmpDir, SecurityRules{})

	err := fs.Write("test.txt", "hello world")
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	content, err := fs.Read("test.txt")
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}
	if content != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", content)
	}
}

func TestFSBridgePathEscape(t *testing.T) {
	tmpDir := t.TempDir()
	fs := newFSBridge(tmpDir, SecurityRules{})

	_, err := fs.Read("../../etc/passwd")
	if err == nil {
		t.Error("expected error for path escape attempt")
	}
	if bridgeErr, ok := err.(*BridgeError); ok {
		if bridgeErr.Code != ErrFSAccessDenied {
			t.Errorf("expected %s, got %s", ErrFSAccessDenied, bridgeErr.Code)
		}
	}
}

func TestFSBridgeExists(t *testing.T) {
	tmpDir := t.TempDir()
	fs := newFSBridge(tmpDir, SecurityRules{})

	exists, err := fs.Exists("nonexistent.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected false for nonexistent file")
	}

	fs.Write("exists.txt", "data")
	exists, err = fs.Exists("exists.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true for existing file")
	}
}

func TestFSBridgeList(t *testing.T) {
	tmpDir := t.TempDir()
	fs := newFSBridge(tmpDir, SecurityRules{})

	fs.Write("a.txt", "a")
	fs.Write("b.txt", "b")

	files, err := fs.List(".")
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestFSBridgeMkdir(t *testing.T) {
	tmpDir := t.TempDir()
	fs := newFSBridge(tmpDir, SecurityRules{})

	err := fs.Mkdir("subdir/nested")
	if err != nil {
		t.Fatalf("failed to mkdir: %v", err)
	}

	info, statErr := os.Stat(filepath.Join(tmpDir, "subdir", "nested"))
	if statErr != nil {
		t.Fatalf("directory not created: %v", statErr)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestFSBridgeRemove(t *testing.T) {
	tmpDir := t.TempDir()
	fs := newFSBridge(tmpDir, SecurityRules{})

	fs.Write("removeme.txt", "data")
	err := fs.Remove("removeme.txt")
	if err != nil {
		t.Fatalf("failed to remove: %v", err)
	}

	exists, _ := fs.Exists("removeme.txt")
	if exists {
		t.Error("file should have been removed")
	}
}

func TestFSBridgeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	fs := newFSBridge(tmpDir, SecurityRules{})

	_, err := fs.Read("nonexistent.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if bridgeErr, ok := err.(*BridgeError); ok {
		if bridgeErr.Code != ErrFSNotFound {
			t.Errorf("expected %s, got %s", ErrFSNotFound, bridgeErr.Code)
		}
	}
}

func TestSecurityRulesMatchesAny(t *testing.T) {
	tests := []struct {
		value    string
		patterns []string
		expected bool
	}{
		{"STRIPE_API_KEY", []string{"STRIPE_*"}, true},
		{"SMTP_HOST", []string{"STRIPE_*"}, false},
		{"CRM_KEY", []string{"CRM_*", "HRM_*"}, true},
		{"OTHER", []string{"CRM_*", "HRM_*"}, false},
		{"ANYTHING", []string{"*"}, true},
	}

	for _, tt := range tests {
		result := matchesAny(tt.value, tt.patterns)
		if result != tt.expected {
			t.Errorf("matchesAny(%q, %v) = %v, want %v", tt.value, tt.patterns, result, tt.expected)
		}
	}
}

func TestCacheBridge(t *testing.T) {
	cache := newCacheBridge(&mockCache{data: make(map[string]any)})

	err := cache.Set("key1", "value1", &CacheSetOptions{TTL: 60})
	if err != nil {
		t.Fatalf("failed to set: %v", err)
	}

	val, err := cache.Get("key1")
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if val != "value1" {
		t.Errorf("expected 'value1', got '%v'", val)
	}

	err = cache.Del("key1")
	if err != nil {
		t.Fatalf("failed to del: %v", err)
	}

	val, err = cache.Get("key1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil after delete, got '%v'", val)
	}
}

func TestLogBridge(t *testing.T) {
	logger := newLogBridge("test_module")
	logger.Log("info", "test message", map[string]any{"key": "value"})
	logger.Log("error", "error message")
	logger.Log("debug", "debug message")
	logger.Log("warn", "warn message")
	logger.Log("unknown", "unknown level")
}

func TestDefaultExecutionLogConfig(t *testing.T) {
	cfg := DefaultExecutionLogConfig()
	if !cfg.Enabled {
		t.Error("expected enabled=true")
	}
	if !cfg.SaveInput {
		t.Error("expected save_input=true")
	}
	if cfg.MaxAge != "30d" {
		t.Errorf("expected max_age=30d, got %s", cfg.MaxAge)
	}
	if cfg.MaxRecords != 100000 {
		t.Errorf("expected max_records=100000, got %d", cfg.MaxRecords)
	}
	if cfg.MaxInputSize != 10240 {
		t.Errorf("expected max_input_size=10240, got %d", cfg.MaxInputSize)
	}
}

func TestTruncateJSON(t *testing.T) {
	small := map[string]any{"key": "value"}
	result := truncateJSON(small, 10240)
	if _, ok := result.(map[string]any); !ok {
		t.Error("expected map for small data")
	}

	result = truncateJSON(nil, 10240)
	if result != nil {
		t.Error("expected nil for nil input")
	}
}

func TestSearchOptions(t *testing.T) {
	opts := SearchOptions{
		Domain:  [][]any{{"status", "=", "new"}},
		Fields:  []string{"name", "email"},
		Order:   "created_at desc",
		Limit:   50,
		Offset:  10,
		Include: []string{"users"},
	}

	q := buildQuery(opts)
	if q == nil {
		t.Fatal("expected non-nil query")
	}
	if q.Offset != 10 {
		t.Errorf("expected offset=10, got %d", q.Offset)
	}
}

func TestBuildQueryEmpty(t *testing.T) {
	q := buildQuery(SearchOptions{})
	if q == nil {
		t.Fatal("expected non-nil query for empty opts")
	}
}

func TestBuildQueryOrderAscDefault(t *testing.T) {
	q := buildQuery(SearchOptions{Order: "name"})
	if q == nil {
		t.Fatal("expected non-nil query")
	}
}

type mockCache struct {
	data map[string]any
}

func (m *mockCache) Get(key string) (any, bool) {
	v, ok := m.data[key]
	return v, ok
}

func (m *mockCache) Set(key string, value any, _ time.Duration) {
	m.data[key] = value
}

func (m *mockCache) Delete(key string) {
	delete(m.data, key)
}

func (m *mockCache) Clear() {
	m.data = make(map[string]any)
}
