package cache

import (
	"testing"
	"time"
)

func TestMemoryCache_SetAndGet(t *testing.T) {
	c := NewMemoryCache()
	c.Set("key1", "value1", 0)

	val, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected key1 to exist")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}
}

func TestMemoryCache_GetMissing(t *testing.T) {
	c := NewMemoryCache()
	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent key to not exist")
	}
}

func TestMemoryCache_TTLExpiry(t *testing.T) {
	c := NewMemoryCache()
	c.Set("key1", "value1", 50*time.Millisecond)

	val, ok := c.Get("key1")
	if !ok || val != "value1" {
		t.Fatal("expected key1 to exist before expiry")
	}

	time.Sleep(60 * time.Millisecond)

	_, ok = c.Get("key1")
	if ok {
		t.Error("expected key1 to be expired")
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	c := NewMemoryCache()
	c.Set("key1", "value1", 0)
	c.Delete("key1")

	_, ok := c.Get("key1")
	if ok {
		t.Error("expected key1 to be deleted")
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	c := NewMemoryCache()
	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Clear()

	_, ok1 := c.Get("a")
	_, ok2 := c.Get("b")
	if ok1 || ok2 {
		t.Error("expected all keys to be cleared")
	}
}
