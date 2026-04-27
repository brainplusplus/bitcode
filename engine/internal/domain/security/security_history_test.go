package security

import (
	"strings"
	"testing"
)

func TestSecurityHistory_NewCreate(t *testing.T) {
	snapshot := map[string]any{
		"name":    "sales.user",
		"display": "Sales User",
	}

	h := NewSecurityHistory("sh-1", "group", "g-1", "sales.user", "create", nil, snapshot, "u-1", "admin", "base")

	if h.EntityType != "group" {
		t.Errorf("expected entity_type group, got %s", h.EntityType)
	}
	if h.EntityID != "g-1" {
		t.Errorf("expected entity_id g-1, got %s", h.EntityID)
	}
	if h.EntityName != "sales.user" {
		t.Errorf("expected entity_name sales.user, got %s", h.EntityName)
	}
	if h.Action != "create" {
		t.Errorf("expected action create, got %s", h.Action)
	}
	if h.Changes != "" {
		t.Errorf("expected empty changes for nil input, got %s", h.Changes)
	}
	if !strings.Contains(h.Snapshot, "sales.user") {
		t.Errorf("snapshot should contain sales.user, got %s", h.Snapshot)
	}
	if h.UserID != "u-1" {
		t.Errorf("expected user_id u-1, got %s", h.UserID)
	}
	if h.Source != "admin" {
		t.Errorf("expected source admin, got %s", h.Source)
	}
	if h.Module != "base" {
		t.Errorf("expected module base, got %s", h.Module)
	}
}

func TestSecurityHistory_NewUpdate(t *testing.T) {
	changes := map[string]any{
		"can_read":  true,
		"can_write": true,
	}
	snapshot := map[string]any{
		"name":      "contact_access",
		"can_read":  true,
		"can_write": true,
	}

	h := NewSecurityHistory("sh-2", "model_access", "ma-1", "contact_access", "update", changes, snapshot, "u-2", "api", "crm")

	if h.Action != "update" {
		t.Errorf("expected action update, got %s", h.Action)
	}
	if !strings.Contains(h.Changes, "can_read") {
		t.Errorf("changes should contain can_read, got %s", h.Changes)
	}
	if !strings.Contains(h.Changes, "can_write") {
		t.Errorf("changes should contain can_write, got %s", h.Changes)
	}
	if !strings.Contains(h.Snapshot, "contact_access") {
		t.Errorf("snapshot should contain contact_access, got %s", h.Snapshot)
	}
}
