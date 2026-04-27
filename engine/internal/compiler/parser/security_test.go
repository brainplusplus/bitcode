package parser

import (
	"strings"
	"testing"
)

func TestParseSecurity_BasicGroup(t *testing.T) {
	data := []byte(`{
		"name": "sales.user",
		"label": "Sales User",
		"category": "sales",
		"implies": ["base.user"],
		"share": false,
		"access": {
			"contact": ["select", "read", "write", "create"],
			"order": ["select", "read"]
		},
		"rules": [
			{
				"name": "own_contacts",
				"model": "contact",
				"domain": [["created_by", "=", "{{user.id}}"]],
				"perm_read": true,
				"perm_write": true,
				"perm_create": true,
				"perm_delete": false
			}
		],
		"menus": ["sales.main_menu"],
		"pages": ["sales.dashboard"],
		"comment": "Basic sales user access"
	}`)

	sec, err := ParseSecurity(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sec.Name != "sales.user" {
		t.Errorf("expected name sales.user, got %s", sec.Name)
	}
	if sec.Label != "Sales User" {
		t.Errorf("expected label Sales User, got %s", sec.Label)
	}
	if sec.Category != "sales" {
		t.Errorf("expected category sales, got %s", sec.Category)
	}
	if len(sec.Implies) != 1 || sec.Implies[0] != "base.user" {
		t.Errorf("expected implies [base.user], got %v", sec.Implies)
	}
	if sec.Share {
		t.Error("expected share=false")
	}
	if len(sec.Access["contact"]) != 4 {
		t.Errorf("expected 4 contact permissions, got %d", len(sec.Access["contact"]))
	}
	if len(sec.Access["order"]) != 2 {
		t.Errorf("expected 2 order permissions, got %d", len(sec.Access["order"]))
	}
	if len(sec.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(sec.Rules))
	}
	rule := sec.Rules[0]
	if rule.Name != "own_contacts" {
		t.Errorf("expected rule name own_contacts, got %s", rule.Name)
	}
	if rule.Model != "contact" {
		t.Errorf("expected rule model contact, got %s", rule.Model)
	}
	if !rule.IsPermRead() {
		t.Error("expected perm_read=true")
	}
	if !rule.IsPermWrite() {
		t.Error("expected perm_write=true")
	}
	if !rule.IsPermCreate() {
		t.Error("expected perm_create=true")
	}
	if rule.IsPermDelete() {
		t.Error("expected perm_delete=false")
	}
	if len(sec.Menus) != 1 || sec.Menus[0] != "sales.main_menu" {
		t.Errorf("expected menus [sales.main_menu], got %v", sec.Menus)
	}
	if len(sec.Pages) != 1 || sec.Pages[0] != "sales.dashboard" {
		t.Errorf("expected pages [sales.dashboard], got %v", sec.Pages)
	}
	if sec.Comment != "Basic sales user access" {
		t.Errorf("expected comment, got %s", sec.Comment)
	}
}

func TestParseSecurity_AllShorthand(t *testing.T) {
	data := []byte(`{
		"name": "admin",
		"access": {
			"contact": "all"
		}
	}`)

	sec, err := ParseSecurity(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	perms := sec.Access["contact"]
	if len(perms) != 12 {
		t.Fatalf("expected 12 permissions from 'all', got %d", len(perms))
	}
	expected := map[string]bool{
		"select": true, "read": true, "write": true, "create": true,
		"delete": true, "export": true, "import": true, "print": true,
		"email": true, "share": true, "report": true, "submit": true,
	}
	for _, p := range perms {
		if !expected[p] {
			t.Errorf("unexpected permission %q", p)
		}
	}
}

func TestParseSecurity_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{
			name:    "missing name",
			json:    `{"label": "No Name"}`,
			wantErr: "security group name is required",
		},
		{
			name:    "invalid permission name",
			json:    `{"name": "test", "access": {"contact": ["read", "fly"]}}`,
			wantErr: "invalid permission \"fly\"",
		},
		{
			name:    "rule without model",
			json:    `{"name": "test", "rules": [{"perm_read": true}]}`,
			wantErr: "rule 0 must specify a model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSecurity([]byte(tt.json))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestParseSecurity_RuleDefaults(t *testing.T) {
	data := []byte(`{
		"name": "viewer",
		"rules": [
			{"model": "contact"}
		]
	}`)

	sec, err := ParseSecurity(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(sec.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(sec.Rules))
	}
	rule := sec.Rules[0]
	if !rule.IsPermRead() {
		t.Error("expected perm_read default true")
	}
	if !rule.IsPermWrite() {
		t.Error("expected perm_write default true")
	}
	if !rule.IsPermCreate() {
		t.Error("expected perm_create default true")
	}
	if !rule.IsPermDelete() {
		t.Error("expected perm_delete default true")
	}
}
