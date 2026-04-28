package stdlib

import (
	"strings"
	"testing"
)

func TestRegexMatch(t *testing.T) {
	r := DefaultRegistry()
	env := r.EnvVars()
	regex := env["regex"].(map[string]any)
	matchFn := regex["match"].(func(...any) (any, error))

	result, err := matchFn("hello@world.com", `^[a-z]+@[a-z]+\.[a-z]+$`)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if result != true {
		t.Error("expected match")
	}

	result, err = matchFn("invalid", `^[0-9]+$`)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if result != false {
		t.Error("expected no match")
	}
}

func TestRegexMatch_InvalidPattern(t *testing.T) {
	r := DefaultRegistry()
	env := r.EnvVars()
	regex := env["regex"].(map[string]any)
	matchFn := regex["match"].(func(...any) (any, error))

	_, err := matchFn("test", `(`)
	if err == nil {
		t.Error("expected error for invalid pattern")
	}
	if !strings.Contains(err.Error(), "invalid pattern") {
		t.Errorf("expected 'invalid pattern' in error, got: %s", err.Error())
	}
}

func TestRegexFindAll(t *testing.T) {
	r := DefaultRegistry()
	env := r.EnvVars()
	regex := env["regex"].(map[string]any)
	findAllFn := regex["findAll"].(func(...any) (any, error))

	result, err := findAllFn("abc123def456", `\d+`)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	arr := result.([]any)
	if len(arr) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(arr))
	}
	if arr[0] != "123" || arr[1] != "456" {
		t.Errorf("unexpected matches: %v", arr)
	}
}

func TestRegexFindAll_NoMatches(t *testing.T) {
	r := DefaultRegistry()
	env := r.EnvVars()
	regex := env["regex"].(map[string]any)
	findAllFn := regex["findAll"].(func(...any) (any, error))

	result, err := findAllFn("hello", `\d+`)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	arr := result.([]any)
	if len(arr) != 0 {
		t.Errorf("expected 0 matches, got %d", len(arr))
	}
}

func TestRegexReplace(t *testing.T) {
	r := DefaultRegistry()
	env := r.EnvVars()
	regex := env["regex"].(map[string]any)
	replaceFn := regex["replace"].(func(...any) (any, error))

	result, err := replaceFn("hello   world", `\s+`, " ")
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %q", result)
	}
}

func TestRegexReplace_NoMatch(t *testing.T) {
	r := DefaultRegistry()
	env := r.EnvVars()
	regex := env["regex"].(map[string]any)
	replaceFn := regex["replace"].(func(...any) (any, error))

	result, err := replaceFn("hello", `\d+`, "X")
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if result != "hello" {
		t.Errorf("expected 'hello' (unchanged), got %q", result)
	}
}

func TestRegexPatternTooLong(t *testing.T) {
	r := DefaultRegistry()
	env := r.EnvVars()
	regex := env["regex"].(map[string]any)
	matchFn := regex["match"].(func(...any) (any, error))

	longPattern := strings.Repeat("a", 1001)
	_, err := matchFn("test", longPattern)
	if err == nil {
		t.Error("expected error for pattern too long")
	}
	if !strings.Contains(err.Error(), "pattern too long") {
		t.Errorf("expected 'pattern too long' in error, got: %s", err.Error())
	}
}

func TestRegexInputTooLarge(t *testing.T) {
	r := DefaultRegistry()
	env := r.EnvVars()
	regex := env["regex"].(map[string]any)
	matchFn := regex["match"].(func(...any) (any, error))

	largeInput := strings.Repeat("a", 1024*1024+1)
	_, err := matchFn(largeInput, "a")
	if err == nil {
		t.Error("expected error for input too large")
	}
	if !strings.Contains(err.Error(), "input too large") {
		t.Errorf("expected 'input too large' in error, got: %s", err.Error())
	}
}

func TestRegexCaching(t *testing.T) {
	r := DefaultRegistry()
	env := r.EnvVars()
	regex := env["regex"].(map[string]any)
	matchFn := regex["match"].(func(...any) (any, error))

	pattern := `^test\d+$`
	matchFn("test123", pattern)
	matchFn("test456", pattern)

	regexCacheMu.RLock()
	_, cached := regexCache[pattern]
	regexCacheMu.RUnlock()

	if !cached {
		t.Error("pattern should be cached after first use")
	}
}

func TestRegexEmptyString(t *testing.T) {
	r := DefaultRegistry()
	env := r.EnvVars()
	regex := env["regex"].(map[string]any)
	matchFn := regex["match"].(func(...any) (any, error))

	result, err := matchFn("", `^$`)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if result != true {
		t.Error("empty string should match ^$")
	}
}
