package setting

import (
	"context"
	"testing"
)

func TestStore_SetAndGet(t *testing.T) {
	s := NewStore()
	s.Set(context.Background(), "app.name", "MyERP")

	val, err := s.Get(context.Background(), "app.name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "MyERP" {
		t.Errorf("expected MyERP, got %s", val)
	}
}

func TestStore_GetNotFound(t *testing.T) {
	s := NewStore()
	_, err := s.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent key")
	}
}

func TestStore_GetWithDefault(t *testing.T) {
	s := NewStore()
	val := s.GetWithDefault("missing", "fallback")
	if val != "fallback" {
		t.Errorf("expected fallback, got %s", val)
	}

	s.Set(context.Background(), "exists", "real")
	val2 := s.GetWithDefault("exists", "fallback")
	if val2 != "real" {
		t.Errorf("expected real, got %s", val2)
	}
}

func TestStore_LoadDefaults(t *testing.T) {
	s := NewStore()
	s.LoadDefaults("sales", map[string]any{
		"default_currency": "USD",
		"tax_rate":         0.1,
	})

	val, _ := s.Get(context.Background(), "sales.default_currency")
	if val != "USD" {
		t.Errorf("expected USD, got %s", val)
	}

	s.Set(context.Background(), "sales.default_currency", "IDR")
	s.LoadDefaults("sales", map[string]any{"default_currency": "USD"})

	val2, _ := s.Get(context.Background(), "sales.default_currency")
	if val2 != "IDR" {
		t.Errorf("expected IDR (should not overwrite), got %s", val2)
	}
}

func TestStore_All(t *testing.T) {
	s := NewStore()
	s.Set(context.Background(), "a", "1")
	s.Set(context.Background(), "b", "2")

	all := s.All()
	if len(all) != 2 {
		t.Errorf("expected 2 settings, got %d", len(all))
	}
}
