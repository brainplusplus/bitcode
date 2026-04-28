package yaegi_runtime

import (
	"testing"
	"time"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
	"github.com/bitcode-framework/bitcode/internal/runtime/embedded"
)

func TestYaegiRuntimeName(t *testing.T) {
	rt := New(nil)
	if rt.Name() != "yaegi" {
		t.Errorf("expected name 'yaegi', got %q", rt.Name())
	}
}

func TestYaegiNewVM(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()
}

func TestYaegiExecuteSimpleScript(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{"input": map[string]any{"name": "test"}}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

func Execute(params map[string]any) (any, error) {
	return map[string]any{"status": "ok"}, nil
}
`
	result, err := vm.Execute(code, "test.go")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", m["status"])
	}
}

func TestYaegiExecuteWithContext(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

import "context"

func Execute(ctx context.Context, params map[string]any) (any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return map[string]any{"ctx_ok": true}, nil
	}
}
`
	result, err := vm.Execute(code, "test_ctx.go")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["ctx_ok"] != true {
		t.Errorf("expected ctx_ok true, got %v", m["ctx_ok"])
	}
}

func TestYaegiExecuteNoParams(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

func Execute() (any, error) {
	return "hello", nil
}
`
	result, err := vm.Execute(code, "test_noparam.go")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}
}

func TestYaegiExecuteReturnsError(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

import "fmt"

func Execute(params map[string]any) (any, error) {
	return nil, fmt.Errorf("something went wrong")
}
`
	_, err = vm.Execute(code, "test_error.go")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "something went wrong" {
		t.Errorf("expected 'something went wrong', got %q", err.Error())
	}
}

func TestYaegiSyntaxError(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main
func Execute( {
`
	_, err = vm.Execute(code, "test_syntax.go")
	if err == nil {
		t.Fatal("expected syntax error, got nil")
	}
}

func TestYaegiMissingExecuteFunc(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

func Helper() string { return "hi" }
`
	_, err = vm.Execute(code, "test_nofunc.go")
	if err == nil {
		t.Fatal("expected error for missing Execute function")
	}
}

func TestYaegiPanicRecovery(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

func Execute(params map[string]any) (any, error) {
	panic("intentional panic")
}
`
	_, err = vm.Execute(code, "test_panic.go")
	if err == nil {
		t.Fatal("expected error from panic, got nil")
	}
}

func TestYaegiTimeout(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 500 * time.Millisecond})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	// Script that cooperatively checks context
	code := `package main

import (
	"context"
	"time"
)

func Execute(ctx context.Context, params map[string]any) (any, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}
`
	_, err = vm.Execute(code, "test_timeout.go")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestYaegiGoroutineSupport(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

import "sync"

func Execute(params map[string]any) (any, error) {
	var mu sync.Mutex
	results := make([]int, 0)
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			mu.Lock()
			results = append(results, n)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	return map[string]any{"count": len(results)}, nil
}
`
	result, err := vm.Execute(code, "test_goroutine.go")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["count"] != 5 {
		t.Errorf("expected count 5, got %v", m["count"])
	}
}

func TestYaegiChannelSupport(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

func Execute(params map[string]any) (any, error) {
	ch := make(chan int, 3)
	ch <- 10
	ch <- 20
	ch <- 30
	close(ch)

	sum := 0
	for v := range ch {
		sum += v
	}
	return map[string]any{"sum": sum}, nil
}
`
	result, err := vm.Execute(code, "test_channel.go")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["sum"] != 60 {
		t.Errorf("expected sum 60, got %v", m["sum"])
	}
}

func TestYaegiStdlibAccess(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func Execute(params map[string]any) (any, error) {
	data := map[string]string{"hello": "world"}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	upper := strings.ToUpper(string(b))
	_ = fmt.Sprintf("test")
	return map[string]any{"json": upper}, nil
}
`
	result, err := vm.Execute(code, "test_stdlib.go")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["json"] == nil {
		t.Error("expected json result")
	}
}

func TestYaegiParamsAccess(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{
		"input":   map[string]any{"name": "Alice"},
		"user_id": "user-123",
	}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

func Execute(params map[string]any) (any, error) {
	input := params["input"].(map[string]any)
	name := input["name"].(string)
	userID := params["user_id"].(string)
	return map[string]any{"name": name, "user_id": userID}, nil
}
`
	result, err := vm.Execute(code, "test_params.go")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["name"] != "Alice" {
		t.Errorf("expected name 'Alice', got %v", m["name"])
	}
	if m["user_id"] != "user-123" {
		t.Errorf("expected user_id 'user-123', got %v", m["user_id"])
	}
}

func TestFilteredStdlibExcludesBlocked(t *testing.T) {
	filtered := FilteredStdlib()

	blockedKeys := []string{
		"os/exec/exec",
		"unsafe/unsafe",
		"syscall/syscall",
		"plugin/plugin",
	}
	for _, key := range blockedKeys {
		if _, exists := filtered[key]; exists {
			t.Errorf("expected %q to be filtered out, but it exists", key)
		}
	}

	allowedKeys := []string{
		"fmt/fmt",
		"strings/strings",
		"encoding/json/json",
		"sync/sync",
		"context/context",
	}
	for _, key := range allowedKeys {
		if _, exists := filtered[key]; !exists {
			t.Errorf("expected %q to be present, but it's missing", key)
		}
	}
}

func TestFilteredStdlibOSExitRemoved(t *testing.T) {
	filtered := FilteredStdlib()

	osSymbols, exists := filtered["os/os"]
	if !exists {
		t.Fatal("expected os/os to be present")
	}

	if _, hasExit := osSymbols["Exit"]; hasExit {
		t.Error("expected os.Exit to be filtered out")
	}

	if _, hasReadFile := osSymbols["ReadFile"]; !hasReadFile {
		t.Error("expected os.ReadFile to be present")
	}

	if _, hasStat := osSymbols["Stat"]; !hasStat {
		t.Error("expected os.Stat to be present")
	}
}

func TestBridgeLoaderEmptyDir(t *testing.T) {
	sources, err := LoadCustomBridges("/nonexistent", nil, "/nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sources) != 0 {
		t.Errorf("expected 0 sources, got %d", len(sources))
	}
}

func TestYaegiInterrupt(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	vm.Interrupt("test interrupt")
}

func TestYaegiBridgeModelCall(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

import "bitcode"

func Execute(params map[string]any) (any, error) {
	record, err := bitcode.Model("contact").Create(map[string]any{
		"name":  "Alice",
		"email": "alice@test.com",
	})
	if err != nil {
		return nil, err
	}
	return record, nil
}
`
	result, err := vm.Execute(code, "test_bridge_model.go")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["name"] != "Alice" {
		t.Errorf("expected name 'Alice', got %v", m["name"])
	}
}

func TestYaegiBridgeLogAndEnv(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

import "bitcode"

func Execute(params map[string]any) (any, error) {
	bitcode.Log("info", "test message")
	val, err := bitcode.Env("APP_KEY")
	if err != nil {
		return nil, err
	}
	cfg := bitcode.Config("app.name")
	translated := bitcode.T("hello")
	return map[string]any{
		"env":        val,
		"config":     cfg,
		"translated": translated,
	}, nil
}
`
	result, err := vm.Execute(code, "test_bridge_log.go")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	_, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
}

func TestYaegiBridgeTxSwapsContext(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

import "bitcode"

func Execute(params map[string]any) (any, error) {
	err := bitcode.Tx(func() error {
		_, err := bitcode.Model("contact").Create(map[string]any{"name": "InTx"})
		return err
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"tx": "ok"}, nil
}
`
	result, err := vm.Execute(code, "test_bridge_tx.go")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["tx"] != "ok" {
		t.Errorf("expected tx 'ok', got %v", m["tx"])
	}
}

func TestYaegiBridgeHTTP(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

import "bitcode"

func Execute(params map[string]any) (any, error) {
	resp, err := bitcode.HTTP().Get("https://example.com")
	if err != nil {
		return nil, err
	}
	return map[string]any{"status": resp.Status}, nil
}
`
	result, err := vm.Execute(code, "test_bridge_http.go")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["status"] != 200 {
		t.Errorf("expected status 200, got %v", m["status"])
	}
}

func TestYaegiBridgeCrypto(t *testing.T) {
	rt := New(nil)
	vm, err := rt.NewVM(embedded.VMOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewVM failed: %v", err)
	}
	defer vm.Close()

	bc := newMockBridgeContext()
	if err := vm.InjectBridge(bc); err != nil {
		t.Fatalf("InjectBridge failed: %v", err)
	}
	if err := vm.InjectParams(map[string]any{}); err != nil {
		t.Fatalf("InjectParams failed: %v", err)
	}

	code := `package main

import "bitcode"

func Execute(params map[string]any) (any, error) {
	hashed, err := bitcode.Crypto().Hash("password123")
	if err != nil {
		return nil, err
	}
	ok, err := bitcode.Crypto().Verify("password123", hashed)
	if err != nil {
		return nil, err
	}
	return map[string]any{"hashed": hashed, "verified": ok}, nil
}
`
	result, err := vm.Execute(code, "test_bridge_crypto.go")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["hashed"] != "hashed" {
		t.Errorf("expected hashed 'hashed', got %v", m["hashed"])
	}
	if m["verified"] != true {
		t.Errorf("expected verified true, got %v", m["verified"])
	}
}

// --- mock bridge context ---

func newMockBridgeContext() *bridge.Context {
	return bridge.NewContext(bridge.ContextDeps{
		TxManager: &mockTxManager{},
		Model:     &mockModelFactory{},
		DB:        &mockDB{},
		HTTP:      &mockHTTP{},
		Cache:     &mockCache{},
		FS:        &mockFS{},
		Session:   bridge.Session{UserID: "test-user"},
		Config:    &mockConfig{},
		Env:       &mockEnv{},
		Emitter:   &mockEmitter{},
		Caller:    &mockCaller{},
		Execer:    &mockExecer{},
		Logger:    &mockLogger{},
		Email:     &mockEmail{},
		Notify:    &mockNotify{},
		Storage:   &mockStorage{},
		I18N:      &mockI18N{},
		Security:  &mockSecurity{},
		Audit:     &mockAudit{},
		Crypto:    &mockCrypto{},
		Execution: &mockExecution{},
	})
}

type mockTxManager struct{}

func (m *mockTxManager) RunTx(parent *bridge.Context, fn func(tx *bridge.Context) error) error {
	return fn(parent)
}

type mockModelFactory struct{}

func (m *mockModelFactory) Model(name string, session bridge.Session, sudo bool) bridge.ModelHandle {
	return &mockModelHandle{}
}

type mockModelHandle struct{}

func (m *mockModelHandle) Search(opts bridge.SearchOptions) ([]map[string]any, error) {
	return nil, nil
}
func (m *mockModelHandle) Get(id string, opts ...bridge.GetOptions) (map[string]any, error) {
	return nil, nil
}
func (m *mockModelHandle) Create(data map[string]any) (map[string]any, error) { return data, nil }
func (m *mockModelHandle) Write(id string, data map[string]any) error         { return nil }
func (m *mockModelHandle) Delete(id string) error                             { return nil }
func (m *mockModelHandle) Count(opts bridge.SearchOptions) (int64, error)     { return 0, nil }
func (m *mockModelHandle) Sum(field string, opts bridge.SearchOptions) (float64, error) {
	return 0, nil
}
func (m *mockModelHandle) Upsert(data map[string]any, uniqueFields []string) (map[string]any, error) {
	return data, nil
}
func (m *mockModelHandle) CreateMany(records []map[string]any) ([]map[string]any, error) {
	return records, nil
}
func (m *mockModelHandle) WriteMany(ids []string, data map[string]any) (*bridge.BulkResult, error) {
	return &bridge.BulkResult{}, nil
}
func (m *mockModelHandle) DeleteMany(ids []string) (*bridge.BulkResult, error) {
	return &bridge.BulkResult{}, nil
}
func (m *mockModelHandle) UpsertMany(records []map[string]any, uniqueFields []string) ([]map[string]any, error) {
	return records, nil
}
func (m *mockModelHandle) AddRelation(id, field string, relatedIDs []string) error    { return nil }
func (m *mockModelHandle) RemoveRelation(id, field string, relatedIDs []string) error { return nil }
func (m *mockModelHandle) SetRelation(id, field string, relatedIDs []string) error    { return nil }
func (m *mockModelHandle) LoadRelation(id, field string) ([]map[string]any, error)    { return nil, nil }
func (m *mockModelHandle) Sudo() bridge.SudoModelHandle                               { return &mockSudoModelHandle{} }

type mockSudoModelHandle struct{ mockModelHandle }

func (m *mockSudoModelHandle) HardDelete(id string) error { return nil }
func (m *mockSudoModelHandle) HardDeleteMany(ids []string) (*bridge.BulkResult, error) {
	return &bridge.BulkResult{}, nil
}
func (m *mockSudoModelHandle) WithTenant(tenantID string) bridge.SudoModelHandle { return m }
func (m *mockSudoModelHandle) SkipValidation() bridge.SudoModelHandle            { return m }
func (m *mockSudoModelHandle) Sudo() bridge.SudoModelHandle                      { return m }

type mockDB struct{}

func (m *mockDB) Query(sql string, args ...any) ([]map[string]any, error) { return nil, nil }
func (m *mockDB) Execute(sql string, args ...any) (*bridge.ExecDBResult, error) {
	return &bridge.ExecDBResult{}, nil
}

type mockHTTP struct{}

func (m *mockHTTP) Get(url string, opts *bridge.HTTPOptions) (*bridge.HTTPResponse, error) {
	return &bridge.HTTPResponse{Status: 200}, nil
}
func (m *mockHTTP) Post(url string, opts *bridge.HTTPOptions) (*bridge.HTTPResponse, error) {
	return &bridge.HTTPResponse{Status: 200}, nil
}
func (m *mockHTTP) Put(url string, opts *bridge.HTTPOptions) (*bridge.HTTPResponse, error) {
	return &bridge.HTTPResponse{Status: 200}, nil
}
func (m *mockHTTP) Patch(url string, opts *bridge.HTTPOptions) (*bridge.HTTPResponse, error) {
	return &bridge.HTTPResponse{Status: 200}, nil
}
func (m *mockHTTP) Delete(url string, opts *bridge.HTTPOptions) (*bridge.HTTPResponse, error) {
	return &bridge.HTTPResponse{Status: 200}, nil
}

type mockCache struct{}

func (m *mockCache) Get(key string) (any, error)                          { return nil, nil }
func (m *mockCache) Set(key string, value any, opts *bridge.CacheSetOptions) error { return nil }
func (m *mockCache) Del(key string) error                                 { return nil }

type mockFS struct{}

func (m *mockFS) Read(path string) (string, error)    { return "", nil }
func (m *mockFS) Write(path, content string) error    { return nil }
func (m *mockFS) Exists(path string) (bool, error)    { return false, nil }
func (m *mockFS) List(path string) ([]string, error)  { return nil, nil }
func (m *mockFS) Mkdir(path string) error             { return nil }
func (m *mockFS) Remove(path string) error            { return nil }

type mockConfig struct{}

func (m *mockConfig) Get(key string) any { return nil }

type mockEnv struct{}

func (m *mockEnv) Get(key string) (string, error) { return "", nil }

type mockEmitter struct{}

func (m *mockEmitter) Emit(event string, data map[string]any) error { return nil }

type mockCaller struct{}

func (m *mockCaller) Call(process string, input map[string]any) (any, error) { return nil, nil }

type mockExecer struct{}

func (m *mockExecer) Exec(cmd string, args []string, opts *bridge.ExecOptions) (*bridge.ExecResult, error) {
	return &bridge.ExecResult{}, nil
}

type mockLogger struct{}

func (m *mockLogger) Log(level, msg string, data ...map[string]any) {}

type mockEmail struct{}

func (m *mockEmail) Send(opts bridge.EmailOptions) error { return nil }

type mockNotify struct{}

func (m *mockNotify) Send(opts bridge.NotifyOptions) error                  { return nil }
func (m *mockNotify) Broadcast(channel string, data map[string]any) error { return nil }

type mockStorage struct{}

func (m *mockStorage) Upload(opts bridge.UploadOptions) (*bridge.Attachment, error) {
	return &bridge.Attachment{}, nil
}
func (m *mockStorage) URL(id string) (string, error)      { return "", nil }
func (m *mockStorage) Download(id string) ([]byte, error) { return nil, nil }
func (m *mockStorage) Delete(id string) error             { return nil }

type mockI18N struct{}

func (m *mockI18N) Translate(locale, key string) string { return key }

type mockSecurity struct{}

func (m *mockSecurity) Permissions(modelName string) (*bridge.ModelPermissions, error) {
	return &bridge.ModelPermissions{}, nil
}
func (m *mockSecurity) HasGroup(groupName string) (bool, error) { return false, nil }
func (m *mockSecurity) Groups() ([]string, error)               { return nil, nil }

type mockAudit struct{}

func (m *mockAudit) Log(opts bridge.AuditOptions) error { return nil }

type mockCrypto struct{}

func (m *mockCrypto) Encrypt(plaintext string) (string, error)     { return plaintext, nil }
func (m *mockCrypto) Decrypt(ciphertext string) (string, error)    { return ciphertext, nil }
func (m *mockCrypto) Hash(value string) (string, error)            { return "hashed", nil }
func (m *mockCrypto) Verify(value, hash string) (bool, error)      { return true, nil }

type mockExecution struct{}

func (m *mockExecution) Search(opts bridge.ExecutionSearchOptions) ([]map[string]any, error) {
	return nil, nil
}
func (m *mockExecution) Get(id string, opts ...bridge.GetOptions) (map[string]any, error) {
	return nil, nil
}
func (m *mockExecution) Current() *bridge.ExecutionInfo { return nil }
func (m *mockExecution) Retry(id string) (map[string]any, error) { return nil, nil }
func (m *mockExecution) Cancel(id string) error { return nil }
