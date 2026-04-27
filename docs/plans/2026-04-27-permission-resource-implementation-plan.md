# Permission System + Convention-Driven Architecture — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace dual Role+Group system with single Odoo-style Group concept (12 ERPNext permissions), add convention-driven auto-CRUD from model JSON, upgrade bc-datatable to permission-aware with modal CRUD support.

**Architecture:** Group is the sole security concept. ModelAccess (12 booleans per model per group) replaces flat Permission strings. RecordRules use m2m groups with Global∩Group composition. Model `"api": true` auto-generates REST endpoints + pages. `securities/*.json` files define groups+ACL+rules per module. Bi-directional JSON↔DB sync with versioning.

**Tech Stack:** Go 1.23+ (Fiber v2, GORM), Stencil.js (TypeScript), SQLite/PostgreSQL/MySQL, JSON definitions.

**Design Doc:** `docs/plans/2026-04-27-permission-resource-architecture-design.md`

---

## Phase 1: Security Domain Entities (Week 1)

### Task 1.1: Create ModelAccess Domain Entity

**Files:**
- Create: `engine/internal/domain/security/model_access.go`
- Create: `engine/internal/domain/security/model_access_test.go`

**Step 1: Write the test**

```go
// engine/internal/domain/security/model_access_test.go
package security

import "testing"

func TestModelAccess_HasPermission(t *testing.T) {
	ma := NewModelAccess("ma-1", "Contact Access", "contact", "grp-1", "crm")
	ma.CanSelect = true
	ma.CanRead = true
	ma.CanWrite = true
	ma.CanCreate = false

	if !ma.HasPermission("select") {
		t.Error("expected select permission")
	}
	if !ma.HasPermission("read") {
		t.Error("expected read permission")
	}
	if ma.HasPermission("create") {
		t.Error("expected no create permission")
	}
	if ma.HasPermission("delete") {
		t.Error("expected no delete permission")
	}
}

func TestModelAccess_AllPermissions(t *testing.T) {
	ma := NewModelAccess("ma-1", "Full Access", "contact", "grp-1", "crm")
	ma.SetAll(true)

	perms := ma.AllPermissions()
	if len(perms) != 12 {
		t.Errorf("expected 12 permissions, got %d", len(perms))
	}
}

func TestModelAccess_SetFromList(t *testing.T) {
	ma := NewModelAccess("ma-1", "Partial", "contact", "grp-1", "crm")
	ma.SetFromList([]string{"select", "read", "write", "export", "clone"})

	if !ma.CanSelect || !ma.CanRead || !ma.CanWrite || !ma.CanExport || !ma.CanClone {
		t.Error("expected listed permissions to be true")
	}
	if ma.CanCreate || ma.CanDelete || ma.CanPrint || ma.CanEmail || ma.CanReport || ma.CanImport || ma.CanMask {
		t.Error("expected unlisted permissions to be false")
	}
}

func TestModelAccess_IsGlobal(t *testing.T) {
	ma := NewModelAccess("ma-1", "Global", "contact", "", "crm")
	if !ma.IsGlobal() {
		t.Error("expected global when group_id is empty")
	}

	ma2 := NewModelAccess("ma-2", "Scoped", "contact", "grp-1", "crm")
	if ma2.IsGlobal() {
		t.Error("expected not global when group_id is set")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd engine && go test ./internal/domain/security/ -run TestModelAccess -v`
Expected: FAIL — `NewModelAccess` not defined

**Step 3: Write implementation**

```go
// engine/internal/domain/security/model_access.go
package security

import (
	"time"

	"github.com/bitcode-framework/bitcode/pkg/ddd"
)

type ModelAccess struct {
	ddd.BaseEntity
	Name           string `json:"name" gorm:"size:200"`
	ModelName      string `json:"model_name" gorm:"size:100;index"`
	GroupID        string `json:"group_id" gorm:"size:100;index"`
	CanSelect      bool   `json:"can_select" gorm:"default:false"`
	CanRead        bool   `json:"can_read" gorm:"default:false"`
	CanWrite       bool   `json:"can_write" gorm:"default:false"`
	CanCreate      bool   `json:"can_create" gorm:"default:false"`
	CanDelete      bool   `json:"can_delete" gorm:"default:false"`
	CanPrint       bool   `json:"can_print" gorm:"default:false"`
	CanEmail       bool   `json:"can_email" gorm:"default:false"`
	CanReport      bool   `json:"can_report" gorm:"default:false"`
	CanExport      bool   `json:"can_export" gorm:"default:false"`
	CanImport      bool   `json:"can_import" gorm:"default:false"`
	CanMask        bool   `json:"can_mask" gorm:"default:false"`
	CanClone       bool   `json:"can_clone" gorm:"default:false"`
	Module         string `json:"module" gorm:"size:100;index"`
	ModifiedSource string `json:"modified_source" gorm:"size:20;default:'json'"`
}

func NewModelAccess(id, name, modelName, groupID, module string) *ModelAccess {
	return &ModelAccess{
		BaseEntity: ddd.BaseEntity{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Name:       name,
		ModelName:  modelName,
		GroupID:    groupID,
		Module:     module,
		ModifiedSource: "json",
	}
}

func (ma *ModelAccess) IsGlobal() bool {
	return ma.GroupID == ""
}

func (ma *ModelAccess) HasPermission(operation string) bool {
	switch operation {
	case "select":
		return ma.CanSelect
	case "read":
		return ma.CanRead
	case "write":
		return ma.CanWrite
	case "create":
		return ma.CanCreate
	case "delete":
		return ma.CanDelete
	case "print":
		return ma.CanPrint
	case "email":
		return ma.CanEmail
	case "report":
		return ma.CanReport
	case "export":
		return ma.CanExport
	case "import":
		return ma.CanImport
	case "mask":
		return ma.CanMask
	case "clone":
		return ma.CanClone
	default:
		return false
	}
}

func (ma *ModelAccess) SetAll(value bool) {
	ma.CanSelect = value
	ma.CanRead = value
	ma.CanWrite = value
	ma.CanCreate = value
	ma.CanDelete = value
	ma.CanPrint = value
	ma.CanEmail = value
	ma.CanReport = value
	ma.CanExport = value
	ma.CanImport = value
	ma.CanMask = value
	ma.CanClone = value
}

func (ma *ModelAccess) SetFromList(perms []string) {
	ma.SetAll(false)
	for _, p := range perms {
		switch p {
		case "select":
			ma.CanSelect = true
		case "read":
			ma.CanRead = true
		case "write":
			ma.CanWrite = true
		case "create":
			ma.CanCreate = true
		case "delete":
			ma.CanDelete = true
		case "print":
			ma.CanPrint = true
		case "email":
			ma.CanEmail = true
		case "report":
			ma.CanReport = true
		case "export":
			ma.CanExport = true
		case "import":
			ma.CanImport = true
		case "mask":
			ma.CanMask = true
		case "clone":
			ma.CanClone = true
		}
	}
}

func (ma *ModelAccess) AllPermissions() []string {
	var perms []string
	if ma.CanSelect { perms = append(perms, "select") }
	if ma.CanRead { perms = append(perms, "read") }
	if ma.CanWrite { perms = append(perms, "write") }
	if ma.CanCreate { perms = append(perms, "create") }
	if ma.CanDelete { perms = append(perms, "delete") }
	if ma.CanPrint { perms = append(perms, "print") }
	if ma.CanEmail { perms = append(perms, "email") }
	if ma.CanReport { perms = append(perms, "report") }
	if ma.CanExport { perms = append(perms, "export") }
	if ma.CanImport { perms = append(perms, "import") }
	if ma.CanMask { perms = append(perms, "mask") }
	if ma.CanClone { perms = append(perms, "clone") }
	return perms
}
```

**Step 4: Run test to verify it passes**

Run: `cd engine && go test ./internal/domain/security/ -run TestModelAccess -v`
Expected: PASS (4 tests)

**Step 5: Commit**

```bash
git add engine/internal/domain/security/model_access.go engine/internal/domain/security/model_access_test.go
git commit -m "feat(security): add ModelAccess domain entity with 12 ERPNext-style permissions"
```

---

### Task 1.2: Create SecurityHistory Domain Entity

**Files:**
- Create: `engine/internal/domain/security/security_history.go`
- Create: `engine/internal/domain/security/security_history_test.go`

**Step 1: Write the test**

```go
// engine/internal/domain/security/security_history_test.go
package security

import (
	"encoding/json"
	"testing"
)

func TestSecurityHistory_NewCreate(t *testing.T) {
	snapshot := map[string]any{"name": "crm.user", "can_read": true}
	h := NewSecurityHistory("h-1", "group", "grp-1", "crm.user", "create", nil, snapshot, "admin-1", "ui", "crm")

	if h.EntityType != "group" {
		t.Errorf("expected entity_type 'group', got %q", h.EntityType)
	}
	if h.Action != "create" {
		t.Errorf("expected action 'create', got %q", h.Action)
	}
	if h.Changes != "" {
		t.Error("expected no changes for create action")
	}

	var snap map[string]any
	if err := json.Unmarshal([]byte(h.Snapshot), &snap); err != nil {
		t.Fatalf("failed to unmarshal snapshot: %v", err)
	}
	if snap["name"] != "crm.user" {
		t.Error("snapshot should contain entity data")
	}
}

func TestSecurityHistory_NewUpdate(t *testing.T) {
	changes := map[string]any{"can_delete": map[string]any{"old": false, "new": true}}
	snapshot := map[string]any{"name": "crm.user", "can_delete": false}
	h := NewSecurityHistory("h-2", "model_access", "ma-1", "crm.user", "update", changes, snapshot, "admin-1", "ui", "crm")

	if h.Action != "update" {
		t.Errorf("expected action 'update', got %q", h.Action)
	}
	if h.Changes == "" {
		t.Error("expected changes for update action")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd engine && go test ./internal/domain/security/ -run TestSecurityHistory -v`
Expected: FAIL

**Step 3: Write implementation**

```go
// engine/internal/domain/security/security_history.go
package security

import (
	"encoding/json"
	"time"

	"github.com/bitcode-framework/bitcode/pkg/ddd"
)

type SecurityHistory struct {
	ddd.BaseEntity
	EntityType string `json:"entity_type" gorm:"size:50;index"`
	EntityID   string `json:"entity_id" gorm:"size:100;index"`
	EntityName string `json:"entity_name" gorm:"size:200"`
	Action     string `json:"action" gorm:"size:20"`
	Changes    string `json:"changes" gorm:"type:text"`
	Snapshot   string `json:"snapshot" gorm:"type:text"`
	UserID     string `json:"user_id" gorm:"size:100;index"`
	Source     string `json:"source" gorm:"size:50"`
	Module     string `json:"module" gorm:"size:100"`
}

func NewSecurityHistory(id, entityType, entityID, entityName, action string, changes, snapshot any, userID, source, module string) *SecurityHistory {
	var changesJSON, snapshotJSON string
	if changes != nil {
		if b, err := json.Marshal(changes); err == nil {
			changesJSON = string(b)
		}
	}
	if snapshot != nil {
		if b, err := json.Marshal(snapshot); err == nil {
			snapshotJSON = string(b)
		}
	}

	return &SecurityHistory{
		BaseEntity: ddd.BaseEntity{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		EntityType: entityType,
		EntityID:   entityID,
		EntityName: entityName,
		Action:     action,
		Changes:    changesJSON,
		Snapshot:   snapshotJSON,
		UserID:     userID,
		Source:     source,
		Module:     module,
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd engine && go test ./internal/domain/security/ -run TestSecurityHistory -v`
Expected: PASS

**Step 5: Commit**

```bash
git add engine/internal/domain/security/security_history.go engine/internal/domain/security/security_history_test.go
git commit -m "feat(security): add SecurityHistory entity for audit trail and rollback"
```

---

### Task 1.3: Upgrade Group Entity

**Files:**
- Modify: `engine/internal/domain/security/group.go`
- Modify: `engine/internal/domain/security/security_test.go` (add new tests)

**Step 1: Write the test**

```go
// Add to engine/internal/domain/security/security_test.go

func TestGroup_ShareGroupCannotImplyNonShare(t *testing.T) {
	shareGroup := NewGroup("g-1", "portal.user", "Portal User", "Portal")
	shareGroup.Share = true

	internalGroup := NewGroup("g-2", "base.user", "Base User", "Base")
	internalGroup.Share = false

	// Share group should not imply non-share group
	// This is a validation rule, not enforced in domain entity directly
	// but the entity should expose the Share field
	if !shareGroup.Share {
		t.Error("expected share group")
	}
	if internalGroup.Share {
		t.Error("expected non-share group")
	}
}

func TestGroup_AllGroupNamesWithShare(t *testing.T) {
	base := NewGroup("g-1", "base.user", "Base User", "Base")
	crm := NewGroup("g-2", "crm.user", "CRM User", "CRM")
	crm.ImpliedGroups = []Group{*base}

	names := crm.AllGroupNames()
	if len(names) != 2 {
		t.Errorf("expected 2 groups, got %d", len(names))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd engine && go test ./internal/domain/security/ -run TestGroup_Share -v`
Expected: FAIL — `Share` field not found

**Step 3: Modify group.go**

```go
// engine/internal/domain/security/group.go
// Add fields to Group struct:

type Group struct {
	ddd.BaseEntity
	Name           string  `json:"name" gorm:"uniqueIndex;size:100"`
	DisplayName    string  `json:"display_name" gorm:"size:200"`
	Category       string  `json:"category" gorm:"size:100;index"`
	Share          bool    `json:"share" gorm:"default:false"`
	Comment        string  `json:"comment" gorm:"type:text"`
	Module         string  `json:"module" gorm:"size:100;index"`
	ModifiedSource string  `json:"modified_source" gorm:"size:20;default:'json'"`
	ImpliedGroups  []Group `json:"implied_groups" gorm:"many2many:group_implies;"`
}
```

Update `NewGroup` to accept module:

```go
func NewGroup(id string, name string, displayName string, category string) *Group {
	return &Group{
		BaseEntity:     ddd.BaseEntity{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Name:           name,
		DisplayName:    displayName,
		Category:       category,
		ModifiedSource: "json",
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd engine && go test ./internal/domain/security/ -v`
Expected: ALL PASS (existing + new tests)

**Step 5: Commit**

```bash
git add engine/internal/domain/security/group.go engine/internal/domain/security/security_test.go
git commit -m "feat(security): upgrade Group entity with share, comment, module, modified_source fields"
```

---

### Task 1.4: Upgrade User Entity — Remove Roles, Add IsSuperuser

**Files:**
- Modify: `engine/internal/domain/security/user.go`
- Modify: `engine/internal/domain/security/security_test.go`

**Step 1: Write the test**

```go
// Add to security_test.go

func TestUser_IsSuperuser(t *testing.T) {
	u, _ := NewUser("u-1", "admin", "admin@test.com", "password")
	u.IsSuperuser = true

	if !u.IsSuperuser {
		t.Error("expected superuser")
	}
}

func TestUser_InGroupWithImplied(t *testing.T) {
	base := NewGroup("g-1", "base.user", "Base User", "Base")
	crm := NewGroup("g-2", "crm.user", "CRM User", "CRM")
	crm.ImpliedGroups = []Group{*base}

	u, _ := NewUser("u-1", "john", "john@test.com", "password")
	u.Groups = []Group{*crm}

	// Direct group check
	if !u.InGroup("crm.user") {
		t.Error("expected user in crm.user")
	}

	// AllGroupNames should resolve implied
	allGroups := u.AllGroupNames()
	found := false
	for _, g := range allGroups {
		if g == "base.user" {
			found = true
		}
	}
	if !found {
		t.Error("expected base.user via implied chain")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd engine && go test ./internal/domain/security/ -run TestUser_Is -v`
Expected: FAIL — `IsSuperuser` not found, `AllGroupNames` not on User

**Step 3: Modify user.go**

```go
// engine/internal/domain/security/user.go
// Changes:
// 1. Add IsSuperuser field
// 2. Remove Roles field and HasPermission via roles
// 3. Add AllGroupNames() that resolves implied groups

type User struct {
	ddd.BaseAggregate
	Username     string    `json:"username" gorm:"uniqueIndex;size:100"`
	Email        string    `json:"email" gorm:"uniqueIndex;size:255"`
	PasswordHash string    `json:"-" gorm:"size:255"`
	Active       bool      `json:"active" gorm:"default:true"`
	IsSuperuser  bool      `json:"is_superuser" gorm:"default:false"`
	LastLogin    time.Time `json:"last_login,omitempty"`
	Groups       []Group   `json:"groups" gorm:"many2many:user_groups;"`
	// REMOVED: Roles []Role `json:"roles" gorm:"many2many:user_roles;"`
}

// REMOVED: func (u *User) HasPermission(permission string) bool { ... }
// Permission checking now goes through ModelAccess, not User directly

func (u *User) AllGroupNames() []string {
	seen := make(map[string]bool)
	for _, g := range u.Groups {
		g.collectGroups(seen)
	}
	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	return result
}

// Keep existing: InGroup, GroupNames, CheckPassword, SetPassword, Activate, Deactivate, RoleNames (deprecated)
```

**Step 4: Run tests**

Run: `cd engine && go test ./internal/domain/security/ -v`
Expected: PASS — but some existing tests referencing `HasPermission` or `Roles` will fail. Fix those:
- `TestUser_HasPermission` → REMOVE (permission check moves to PermissionChecker service)
- Any test referencing `u.Roles` → REMOVE or update

**Step 5: Fix broken tests, then commit**

```bash
git add engine/internal/domain/security/user.go engine/internal/domain/security/security_test.go
git commit -m "feat(security): upgrade User entity — add IsSuperuser, remove Roles, add AllGroupNames"
```

---

### Task 1.5: Upgrade RecordRule — M2M Groups

**Files:**
- Modify: `engine/internal/domain/security/record_rule.go`
- Modify: `engine/internal/domain/security/security_test.go`

**Step 1: Write the test**

```go
func TestRecordRule_AppliesToGroupM2M(t *testing.T) {
	g1 := NewGroup("g-1", "crm.user", "CRM User", "CRM")
	g2 := NewGroup("g-2", "sales.user", "Sales User", "Sales")

	rule := &RecordRule{
		Name:      "own_contacts",
		ModelName: "contact",
		Groups:    []Group{*g1},
		CanRead:   true,
		CanWrite:  true,
		Active:    true,
	}

	// User in crm.user → rule applies
	if !rule.AppliesToGroupNames([]string{"crm.user"}) {
		t.Error("expected rule to apply to crm.user")
	}

	// User in sales.user → rule does not apply
	if rule.AppliesToGroupNames([]string{"sales.user"}) {
		t.Error("expected rule NOT to apply to sales.user")
	}
}

func TestRecordRule_GlobalRule(t *testing.T) {
	rule := &RecordRule{
		Name:      "active_only",
		ModelName: "contact",
		Groups:    nil, // no groups = global
		CanRead:   true,
		Active:    true,
	}

	if !rule.IsGlobal() {
		t.Error("expected global rule when no groups")
	}

	// Global rule applies to everyone
	if !rule.AppliesToGroupNames([]string{"any.group"}) {
		t.Error("expected global rule to apply to any group")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd engine && go test ./internal/domain/security/ -run TestRecordRule_Applies -v`
Expected: FAIL

**Step 3: Modify record_rule.go**

```go
// engine/internal/domain/security/record_rule.go
// Changes:
// 1. Replace GroupNames string with Groups []Group m2m
// 2. Update AppliesToGroup to use m2m
// 3. Keep backward compat with GroupNames for migration period

type RecordRule struct {
	ddd.BaseEntity
	Name           string  `json:"name" gorm:"uniqueIndex;size:100"`
	ModelName      string  `json:"model_name" gorm:"size:100;index"`
	Groups         []Group `json:"groups" gorm:"many2many:record_rule_groups;"`
	GroupNames     string  `json:"group_names" gorm:"size:500"` // DEPRECATED — kept for migration
	DomainFilter   string  `json:"domain_filter" gorm:"type:text"`
	CanRead        bool    `json:"can_read" gorm:"default:true"`
	CanCreate      bool    `json:"can_create" gorm:"default:true"`
	CanWrite       bool    `json:"can_write" gorm:"default:true"`
	CanDelete      bool    `json:"can_delete" gorm:"default:false"`
	Global         bool    `json:"global" gorm:"default:false"`
	Active         bool    `json:"active" gorm:"default:true"`
	Module         string  `json:"module" gorm:"size:100"`
	ModifiedSource string  `json:"modified_source" gorm:"size:20;default:'json'"`
}

func (r *RecordRule) IsGlobal() bool {
	return len(r.Groups) == 0 && r.GroupNames == ""
}

func (r *RecordRule) AppliesToGroupNames(userGroups []string) bool {
	if r.IsGlobal() {
		return true
	}
	// Check m2m groups first
	for _, rg := range r.Groups {
		for _, ug := range userGroups {
			if rg.Name == ug {
				return true
			}
		}
	}
	// Fallback to legacy GroupNames string
	if r.GroupNames != "" {
		ruleGroups := strings.Split(r.GroupNames, ",")
		for _, rg := range ruleGroups {
			for _, ug := range userGroups {
				if strings.TrimSpace(rg) == ug {
					return true
				}
			}
		}
	}
	return false
}
```

**Step 4: Run tests**

Run: `cd engine && go test ./internal/domain/security/ -v`
Expected: ALL PASS. Fix any existing tests that break due to struct changes.

**Step 5: Commit**

```bash
git add engine/internal/domain/security/record_rule.go engine/internal/domain/security/security_test.go
git commit -m "feat(security): upgrade RecordRule to m2m groups, add module/modified_source"
```

---

### Task 1.6: Create Security JSON Parser

**Files:**
- Create: `engine/internal/compiler/parser/security.go`
- Create: `engine/internal/compiler/parser/security_test.go`

**Step 1: Write the test**

```go
// engine/internal/compiler/parser/security_test.go
package parser

import "testing"

func TestParseSecurity_BasicGroup(t *testing.T) {
	data := []byte(`{
		"name": "crm.user",
		"label": "CRM / User",
		"category": "CRM",
		"implies": ["base.user"],
		"share": false,
		"access": {
			"contact": ["select", "read", "write", "create", "export", "clone"],
			"lead": "all"
		},
		"rules": [
			{
				"name": "crm_user_own_contacts",
				"model": "contact",
				"domain": [["created_by", "=", "{{user.id}}"]],
				"perm_read": true,
				"perm_write": true,
				"perm_create": true,
				"perm_delete": false
			}
		],
		"menus": ["crm/contacts", "crm/leads"],
		"pages": ["contact_list", "contact_form"],
		"comment": "Basic CRM access"
	}`)

	sec, err := ParseSecurity(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sec.Name != "crm.user" {
		t.Errorf("expected name 'crm.user', got %q", sec.Name)
	}
	if sec.Category != "CRM" {
		t.Errorf("expected category 'CRM', got %q", sec.Category)
	}
	if len(sec.Implies) != 1 || sec.Implies[0] != "base.user" {
		t.Errorf("expected implies [base.user], got %v", sec.Implies)
	}
	if sec.Share {
		t.Error("expected share=false")
	}

	// Access
	if len(sec.Access) != 2 {
		t.Fatalf("expected 2 access entries, got %d", len(sec.Access))
	}
	contactAccess := sec.Access["contact"]
	if len(contactAccess) != 6 {
		t.Errorf("expected 6 contact permissions, got %d", len(contactAccess))
	}
	leadAccess := sec.Access["lead"]
	if len(leadAccess) != 12 {
		t.Errorf("expected 12 lead permissions (all), got %d", len(leadAccess))
	}

	// Rules
	if len(sec.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(sec.Rules))
	}
	if sec.Rules[0].Model != "contact" {
		t.Errorf("expected rule model 'contact', got %q", sec.Rules[0].Model)
	}

	// Menus & Pages
	if len(sec.Menus) != 2 {
		t.Errorf("expected 2 menus, got %d", len(sec.Menus))
	}
	if len(sec.Pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(sec.Pages))
	}
}

func TestParseSecurity_ValidationErrors(t *testing.T) {
	// Missing name
	_, err := ParseSecurity([]byte(`{"label": "Test"}`))
	if err == nil {
		t.Error("expected error for missing name")
	}

	// Invalid JSON
	_, err = ParseSecurity([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd engine && go test ./internal/compiler/parser/ -run TestParseSecurity -v`
Expected: FAIL

**Step 3: Write implementation**

```go
// engine/internal/compiler/parser/security.go
package parser

import (
	"encoding/json"
	"fmt"
)

var allPermissions = []string{
	"select", "read", "write", "create", "delete",
	"print", "email", "report",
	"export", "import", "mask", "clone",
}

type SecurityDefinition struct {
	Name    string                  `json:"name"`
	Label   string                  `json:"label,omitempty"`
	Category string                 `json:"category,omitempty"`
	Implies []string                `json:"implies,omitempty"`
	Share   bool                    `json:"share,omitempty"`
	Access  map[string]SecurityACL  `json:"access,omitempty"`
	Rules   []SecurityRuleDefinition `json:"rules,omitempty"`
	Menus   []string                `json:"menus,omitempty"`
	Pages   []string                `json:"pages,omitempty"`
	Comment string                  `json:"comment,omitempty"`
}

// SecurityACL can be either "all" (string) or ["select","read",...] (array)
type SecurityACL []string

func (a *SecurityACL) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s == "all" {
			*a = make([]string, len(allPermissions))
			copy(*a, allPermissions)
			return nil
		}
		return fmt.Errorf("invalid access shorthand: %q (only 'all' is supported)", s)
	}

	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*a = arr
		return nil
	}

	return fmt.Errorf("access must be 'all' or array of permission strings")
}

type SecurityRuleDefinition struct {
	Name       string  `json:"name"`
	Model      string  `json:"model"`
	Domain     [][]any `json:"domain,omitempty"`
	PermRead   *bool   `json:"perm_read,omitempty"`
	PermWrite  *bool   `json:"perm_write,omitempty"`
	PermCreate *bool   `json:"perm_create,omitempty"`
	PermDelete *bool   `json:"perm_delete,omitempty"`
	Global     bool    `json:"global,omitempty"`
}

func (r *SecurityRuleDefinition) IsPermRead() bool {
	if r.PermRead == nil { return true }
	return *r.PermRead
}
func (r *SecurityRuleDefinition) IsPermWrite() bool {
	if r.PermWrite == nil { return true }
	return *r.PermWrite
}
func (r *SecurityRuleDefinition) IsPermCreate() bool {
	if r.PermCreate == nil { return true }
	return *r.PermCreate
}
func (r *SecurityRuleDefinition) IsPermDelete() bool {
	if r.PermDelete == nil { return true }
	return *r.PermDelete
}

func ParseSecurity(data []byte) (*SecurityDefinition, error) {
	var sec SecurityDefinition
	if err := json.Unmarshal(data, &sec); err != nil {
		return nil, fmt.Errorf("invalid security JSON: %w", err)
	}
	if sec.Name == "" {
		return nil, fmt.Errorf("security definition must have a name")
	}
	// Validate access permissions
	for model, perms := range sec.Access {
		for _, p := range perms {
			valid := false
			for _, ap := range allPermissions {
				if p == ap { valid = true; break }
			}
			if !valid {
				return nil, fmt.Errorf("invalid permission %q for model %q", p, model)
			}
		}
	}
	// Validate rules
	for i, rule := range sec.Rules {
		if rule.Model == "" {
			return nil, fmt.Errorf("rule %d must have a model", i)
		}
	}
	return &sec, nil
}

func ParseSecurityFile(path string) (*SecurityDefinition, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return ParseSecurity(data)
}
```

**Step 4: Run tests**

Run: `cd engine && go test ./internal/compiler/parser/ -run TestParseSecurity -v`
Expected: PASS

**Step 5: Commit**

```bash
git add engine/internal/compiler/parser/security.go engine/internal/compiler/parser/security_test.go
git commit -m "feat(parser): add security JSON parser for securities/*.json files"
```

---

### Task 1.7: Upgrade Model Parser — Add API, Mask, Groups Fields

**Files:**
- Modify: `engine/internal/compiler/parser/model.go`
- Modify: `engine/internal/compiler/parser/model_test.go`

**Step 1: Write the test**

```go
// Add to model_test.go

func TestParseModel_WithAPIConfig(t *testing.T) {
	data := []byte(`{
		"name": "contact",
		"module": "crm",
		"fields": {
			"name": { "type": "string", "required": true }
		},
		"api": {
			"auto_crud": true,
			"auth": true,
			"auto_pages": true,
			"modal": false,
			"protocols": { "rest": true, "graphql": true, "websocket": false },
			"search": ["name"],
			"soft_delete": true
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if model.API == nil {
		t.Fatal("expected API config")
	}
	if !model.API.AutoCRUD {
		t.Error("expected auto_crud=true")
	}
	if !model.API.Protocols.GraphQL {
		t.Error("expected graphql=true")
	}
	if model.API.Modal {
		t.Error("expected modal=false")
	}
}

func TestParseModel_WithAPIShorthand(t *testing.T) {
	data := []byte(`{
		"name": "tag",
		"module": "crm",
		"fields": { "name": { "type": "string" } },
		"api": true
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if model.API == nil {
		t.Fatal("expected API config from shorthand")
	}
	if !model.API.AutoCRUD {
		t.Error("expected auto_crud=true from shorthand")
	}
}

func TestParseModel_WithFieldMaskAndGroups(t *testing.T) {
	data := []byte(`{
		"name": "employee",
		"module": "hrm",
		"fields": {
			"name":   { "type": "string" },
			"phone":  { "type": "string", "mask": true, "mask_length": 4 },
			"salary": { "type": "decimal", "groups": ["hr.manager"] }
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	phone := model.Fields["phone"]
	if !phone.Mask {
		t.Error("expected phone mask=true")
	}
	if phone.MaskLength != 4 {
		t.Error("expected phone mask_length=4")
	}
	salary := model.Fields["salary"]
	if len(salary.Groups) != 1 || salary.Groups[0] != "hr.manager" {
		t.Errorf("expected salary groups=[hr.manager], got %v", salary.Groups)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd engine && go test ./internal/compiler/parser/ -run TestParseModel_WithAPI -v`
Expected: FAIL

**Step 3: Add to model.go**

Add to `FieldDefinition` struct:
```go
Mask       bool     `json:"mask,omitempty"`
MaskLength int      `json:"mask_length,omitempty"`
Groups     []string `json:"groups,omitempty"`
```

Add new types and field to `ModelDefinition`:
```go
type APIConfig struct {
	AutoCRUD  bool            `json:"auto_crud"`
	Auth      bool            `json:"auth"`
	AutoPages json.RawMessage `json:"auto_pages,omitempty"` // bool or object
	Modal     bool            `json:"modal,omitempty"`
	Protocols ProtocolConfig  `json:"protocols,omitempty"`
	Search    []string        `json:"search,omitempty"`
	SoftDelete *bool          `json:"soft_delete,omitempty"`
}

type ProtocolConfig struct {
	REST      bool `json:"rest"`
	GraphQL   bool `json:"graphql"`
	WebSocket bool `json:"websocket"`
}

func (a *APIConfig) IsAutoPages() bool {
	if a.AutoPages == nil { return true }
	var b bool
	if err := json.Unmarshal(a.AutoPages, &b); err == nil { return b }
	return true // if it's an object, auto_pages is enabled (with config)
}

func (a *APIConfig) IsSoftDelete() bool {
	if a.SoftDelete == nil { return true }
	return *a.SoftDelete
}
```

Add to `ModelDefinition`:
```go
APIRaw json.RawMessage `json:"api,omitempty"`
API    *APIConfig       `json:"-"`
```

In `ParseModel()`, after unmarshal, resolve API:
```go
if model.APIRaw != nil {
	// Try bool shorthand first
	var apiBool bool
	if err := json.Unmarshal(model.APIRaw, &apiBool); err == nil {
		if apiBool {
			model.API = &APIConfig{
				AutoCRUD: true, Auth: true,
				Protocols: ProtocolConfig{REST: true},
			}
		}
	} else {
		// Try full object
		var apiConfig APIConfig
		if err := json.Unmarshal(model.APIRaw, &apiConfig); err == nil {
			model.API = &apiConfig
		}
	}
}
```

**Step 4: Run tests**

Run: `cd engine && go test ./internal/compiler/parser/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add engine/internal/compiler/parser/model.go engine/internal/compiler/parser/model_test.go
git commit -m "feat(parser): add api config, field mask/mask_length/groups to model parser"
```

---

### Task 1.8: Upgrade Module Parser — Add Securities, Rename Views to Pages

**Files:**
- Modify: `engine/internal/compiler/parser/module.go`
- Modify: `engine/internal/infrastructure/module/module_test.go`

**Step 1: Write the test**

```go
// Add to module_test.go

func TestParseModule_WithSecurities(t *testing.T) {
	data := []byte(`{
		"name": "crm",
		"version": "1.0.0",
		"depends": ["base"],
		"models": ["models/*.json"],
		"securities": ["securities/*.json"],
		"pages": ["pages/*.json"],
		"menu": [
			{ "label": "Contacts", "page": "contact_list", "groups": ["crm.user"] }
		]
	}`)

	mod, err := parser.ParseModule(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(mod.Securities) != 1 || mod.Securities[0] != "securities/*.json" {
		t.Errorf("expected securities glob, got %v", mod.Securities)
	}
	if len(mod.Pages) != 1 {
		t.Errorf("expected pages glob, got %v", mod.Pages)
	}
}
```

**Step 2: Run test, verify fail**

**Step 3: Add to ModuleDefinition**

```go
Securities []string `json:"securities,omitempty"`
Pages      []string `json:"pages,omitempty"`    // new — replaces Views for new modules
// Views still supported for backward compat
```

Add to `MenuItemDefinition`:
```go
Page   string   `json:"page,omitempty"`   // new — reference to page name
Groups []string `json:"groups,omitempty"` // new — visibility per group
```

**Step 4: Run tests, verify pass**

**Step 5: Commit**

```bash
git commit -m "feat(parser): add securities and pages to module parser, menu groups"
```

---

### Task 1.9: DB Migration — Create New Tables

**Files:**
- Modify: `engine/internal/app.go` (add AutoMigrate calls)

**Step 1: Add AutoMigrate for new entities**

In `app.go` where DB migration happens, add:
```go
db.AutoMigrate(
	&security.ModelAccess{},
	&security.SecurityHistory{},
	// Group already migrated — GORM will add new columns
)
```

**Step 2: Run the app, verify tables created**

Run: `cd engine && go run ./cmd/bitcode/ serve`
Check: `model_access`, `ir_security_histories`, `record_rule_groups` tables exist.
Check: `groups` table has new columns (share, comment, module, modified_source).
Check: `users` table has `is_superuser` column.

**Step 3: Commit**

```bash
git commit -m "feat(db): auto-migrate new security tables (model_access, security_history, record_rule_groups)"
```

---

## Phase 2: Permission Enforcement (Week 2-3)

### Task 2.1: Implement PermissionChecker Service

**Files:**
- Create: `engine/internal/infrastructure/persistence/permission_checker.go`
- Create: `engine/internal/infrastructure/persistence/permission_checker_test.go`

This is the concrete implementation of the `PermissionChecker` interface that the middleware uses.

**Key logic:**
1. Load user's groups (with implied chain resolved)
2. Query `model_access` table for matching group_ids + global (group_id = "")
3. Union all permissions (additive)
4. Return bool for requested operation

**Test cases:**
- User in group with read → can read
- User in group without delete → cannot delete
- User in two groups, one has write → can write (additive)
- Global ACL (group_id="") → applies to all users
- No ACL for model → default deny
- Superuser → always true

### Task 2.2: Implement RecordRuleEngine Service

**Files:**
- Create: `engine/internal/infrastructure/persistence/record_rule_engine.go`
- Create: `engine/internal/infrastructure/persistence/record_rule_engine_test.go`

**Key logic:**
1. Load user's groups
2. Query record_rules for model
3. Separate global vs group rules
4. Global rules → AND (intersect)
5. Group rules → OR (union)
6. Final = AND(global) AND OR(group)
7. Return domain filters for WHERE clause injection

**Test cases:**
- Global rule only → applied as filter
- Group rule only → applied if user in group
- Global + Group → intersected
- Two global rules → both must match (AND)
- Two group rules for same user → either can match (OR)
- Superuser → no filters

### Task 2.3: Wire Permission Middleware in Route Registration

**Files:**
- Modify: `engine/internal/presentation/api/router.go`
- Modify: `engine/internal/app.go`

**Key changes:**
- `RegisterAPI()` now attaches `PermissionMiddleware` per endpoint
- Permission derived from endpoint action: list/read → "read", create → "create", etc.
- `RecordRuleMiddleware` attached for list/read operations

### Task 2.4: CRUD Handler — Field Masking + Field Groups

**Files:**
- Modify: `engine/internal/presentation/api/crud_handler.go`

**Key changes:**
- After query, before response: strip fields where user not in field.groups
- After query, before response: mask fields where field.mask=true and user lacks can_mask
- Masking function: `maskValue(value string, maskLength int) string` → `"****1234"`
- Inject user permissions into response metadata for frontend

### Task 2.5: CRUD Handler — Inject Permissions into Page Context

**Files:**
- Modify: `engine/internal/presentation/api/crud_handler.go`
- Modify: `engine/internal/presentation/view/renderer.go`

**Key changes:**
- When rendering pages, include user's permissions for the model in template context
- `bc-datatable` receives permissions as prop
- Form renderer receives permissions for button visibility

---

## Phase 3: Convention-Driven CRUD (Week 3-4)

### Task 3.1: Auto-Generate REST Endpoints from Model

**Files:**
- Modify: `engine/internal/presentation/api/router.go`
- Modify: `engine/internal/app.go`

**Key changes:**
- If model has `api.auto_crud=true` (or `api=true`), auto-register CRUD routes
- No need for separate `apis/*.json` file
- URL pattern: `/api/v1/{module}/{model_plural}`
- Pluralization: simple `s` suffix (configurable later)

### Task 3.2: API Override Merge Logic

**Files:**
- Modify: `engine/internal/presentation/api/router.go`

**Key changes:**
- Load `apis/*.json` files
- Match by method + path against auto-generated endpoints
- Override matched endpoints, add new ones
- Log overrides: `[INFO] Endpoint PUT /api/v1/crm/contacts/:id overridden by apis/contact_api.json`

### Task 3.3: Auto-Generate Pages from Model

**Files:**
- Create: `engine/internal/presentation/view/auto_page_generator.go`
- Create: `engine/internal/presentation/view/auto_page_generator_test.go`

**Key changes:**
- Generate list ViewDefinition from model fields
- Generate form ViewDefinition from model fields
- Use `bc-datatable` for list (not `bc-view-list`)
- Pass permissions, modal mode, URLs as props

### Task 3.4: Page Override Logic

**Files:**
- Modify: `engine/internal/presentation/view/renderer.go`

**Key changes:**
- Load `pages/*.json` files
- Match by model + type against auto-generated pages
- Override matched pages
- Custom pages (type=custom) added as-is

### Task 3.5: Security Loader — Load securities/*.json, Sync to DB

**Files:**
- Create: `engine/internal/infrastructure/module/security_loader.go`
- Create: `engine/internal/infrastructure/module/security_loader_test.go`

**Key changes:**
- Parse securities/*.json files
- UPSERT groups to DB
- UPSERT model_access entries
- UPSERT record_rules
- Sync group_menus and group_pages
- Respect `modified_source` for noupdate behavior
- Record changes in ir_security_histories

### Task 3.6: Loading Order in app.go

**Files:**
- Modify: `engine/internal/app.go`

**Key changes:**
- Reorder: models → securities → apis → pages → processes → agents → templates → i18n → migrations

### Task 3.7: Rename views → pages Throughout Codebase

**Files:**
- Multiple files across engine and modules

**Key changes:**
- `module.json`: support both `"views"` and `"pages"` (backward compat)
- `loadViews()` → `loadPages()`
- View-related variable names → page-related
- Existing modules: keep `views/` folder working, new modules use `pages/`

---

## Phase 4: Component Upgrades (Week 4-5)

### Task 4.1: bc-datatable — Permission Props

**Files:**
- Modify: `packages/components/src/components/datatable/bc-datatable/bc-datatable.tsx`

**Key changes:**
- Add `permissions` prop (JSON string of 12 booleans)
- Add `createUrl`, `editUrl`, `detailUrl` props
- Add `moduleName` prop
- Parse permissions in `componentWillLoad()`

### Task 4.2: bc-datatable — Permission-Aware Toolbar

**Files:**
- Modify: `packages/components/src/components/datatable/bc-datatable/bc-datatable.tsx`

**Key changes:**
- "New" button: visible only if `can_create`
- "Export" button: visible only if `can_export`
- "Import" button: visible only if `can_import`
- Bulk "Delete": visible only if `can_delete`
- Bulk "Clone": visible only if `can_clone`

### Task 4.3: bc-datatable — Row Actions

**Files:**
- Modify: `packages/components/src/components/datatable/bc-datatable/bc-datatable.tsx`

**Key changes:**
- Add actions column (rightmost)
- Edit icon → visible if `can_write`, navigates to `editUrl`
- Delete icon → visible if `can_delete`
- Clone icon → visible if `can_clone`
- Print icon → visible if `can_print`
- Row click → navigate to `detailUrl` (if `can_read`)

### Task 4.4: bc-datatable — Modal Mode

**Files:**
- Modify: `packages/components/src/components/datatable/bc-datatable/bc-datatable.tsx`

**Key changes:**
- Add `modalMode` prop
- When `modalMode=true`:
  - "New" button → open `bc-dialog-modal` with form
  - Row click → open modal with record data
  - Edit action → open modal in edit mode
  - Save in modal → API call → refresh table → close modal
- Add `formFields` prop for modal form field definitions

### Task 4.5: bc-view-form — Permission Props

**Files:**
- Modify: `packages/components/src/components/views/bc-view-form/bc-view-form.tsx`

**Key changes:**
- Add `permissions` prop
- Save button: visible only if `can_write` (edit) or `can_create` (new)
- Delete button: visible only if `can_delete`
- Clone button: visible only if `can_clone`

### Task 4.6: CompileList — Switch to bc-datatable

**Files:**
- Modify: `engine/internal/presentation/view/component_compiler.go`

**Key changes:**
- `CompileList()` now emits `<bc-datatable>` instead of `<bc-view-list>`
- Pass permissions, module name, URLs, modal mode as props
- Convert view fields to column definitions JSON

---

## Phase 5: Admin UI (Week 5-7)

### Task 5.1: Admin URL — Module Prefix

**Files:**
- Modify: `engine/internal/presentation/admin/admin.go`

**Key changes:**
- `/admin/models/:name` → `/admin/models/:module/:name`
- Update all admin links and routes

### Task 5.2: Model Admin — API Tab

**Files:**
- Modify: `engine/internal/presentation/admin/admin.go`

**Key changes:**
- Add "API" tab to model detail page
- Render API config form (checkboxes, dropdowns)
- Show generated endpoints preview (read-only)
- Save updates model JSON file

### Task 5.3: Model Admin — Fields Table Upgrade

**Files:**
- Modify: `engine/internal/presentation/admin/admin.go`

**Key changes:**
- Add MASK and GROUPS columns to fields table
- MASK: show "✔ /N" format
- GROUPS: show comma-separated group names

### Task 5.4: Group List Page

**Files:**
- Modify: `engine/internal/presentation/admin/admin.go`

**Key changes:**
- New route: `/admin/groups`
- List all groups with columns: Name, Label, Category, Share, Module, Users count, Modified source

### Task 5.5: Group Detail Page — All 7 Tabs

**Files:**
- Modify: `engine/internal/presentation/admin/admin.go`

**Key changes:**
- New route: `/admin/groups/:name`
- 7 tabs: Users, Inherited, Menus, Pages, Access Rights, Record Rules, Notes
- Each tab with CRUD (add/remove/edit)
- Access Rights: 12-column checkbox matrix grouped into Core/Action/Data
- "View Effective Permissions" action button

### Task 5.6: Security Sync Page

**Files:**
- Modify: `engine/internal/presentation/admin/admin.go`
- Create: `engine/internal/presentation/api/security_handler.go`

**Key changes:**
- New route: `/admin/securities`
- Buttons: Load from Files, Export to Files, Upload JSON/ZIP, Download ZIP
- History table with rollback buttons
- Upload endpoint: parse JSON/ZIP, preview diff, apply
- Download endpoint: generate ZIP from DB

---

## Phase 6: CLI + Swagger (Week 7-8)

### Task 6.1: Security CLI Commands

**Files:**
- Modify: `engine/cmd/bitcode/main.go` (or create `engine/cmd/bitcode/security.go`)

**Key changes:**
- `bitcode security load [module] [--force]`
- `bitcode security export [module]`
- `bitcode security diff [module]`
- `bitcode security validate [module]`
- `bitcode security history [--entity=name]`
- `bitcode security rollback <history_id>`

### Task 6.2: CRUD Generate CLI

**Files:**
- Modify: `engine/cmd/bitcode/main.go`

**Key changes:**
- `bitcode publish:crud <module> <model> all|api|pages [list|form]`
- Generate override files from auto-generated snapshot
- Write to apis/ or pages/ folder

### Task 6.3: Auto Swagger/OpenAPI Generation

**Files:**
- Create: `engine/internal/presentation/api/swagger.go`
- Create: `engine/internal/presentation/api/swagger_test.go`

**Key changes:**
- Generate OpenAPI 3.0 spec from model definitions + API definitions
- Fields → schema properties (type mapping)
- Endpoints → paths
- Auth → security schemes
- Serve at `/api/v1/docs/openapi.json`

### Task 6.4: Swagger UI

**Files:**
- Modify: `engine/internal/app.go`

**Key changes:**
- Embed swagger-ui static files (or use CDN)
- Serve at `/api/v1/docs`
- Point to `/api/v1/docs/openapi.json`

### Task 6.5: Update Existing Modules

**Files:**
- Modify: `engine/embedded/modules/base/`
- Modify: `engine/modules/crm/`
- Modify: `engine/modules/sales/`

**Key changes:**
- Add `securities/` folder with group definitions
- Add `api` field to model JSONs
- Move `record_rules` from models to securities
- Rename `views/` to `pages/` (keep backward compat)
- Remove `permissions` and `groups` from `module.json`
- Delete Role/Permission seed data, replace with Group/ModelAccess seeds

### Task 6.6: Delete Old Code

**Files:**
- Delete: `engine/internal/domain/security/role.go`
- Delete: `engine/internal/domain/security/permission.go`
- Modify: Remove role-related code from admin, auth handler, etc.

### Task 6.7: Update Documentation

**Files:**
- Modify: `docs/architecture.md`
- Modify: `docs/features.md`
- Modify: `docs/codebase.md`
- Modify: `engine/docs/features/security.md`
- Create: `engine/docs/features/permissions.md`
- Modify: `AGENTS.md`
- Modify: `README.md`

### Task 6.8: Full Integration Tests

**Files:**
- Create: `engine/internal/presentation/api/permission_integration_test.go`

**Key test scenarios:**
- Module install → groups + ACL + rules synced to DB
- API request without auth → 401
- API request with auth but no ACL → 403
- API request with correct ACL → 200
- List with record rules → filtered results
- Field masking → masked values in response
- Field groups → hidden fields in response
- Superuser → bypass all
- Admin UI → group CRUD works
- Security sync → JSON ↔ DB round-trip

---

## Phase 7: Multi-Protocol (Week 8-10, Optional)

### Task 7.1: GraphQL Schema Generator
### Task 7.2: GraphQL Resolvers
### Task 7.3: WebSocket CRUD Protocol
### Task 7.4: Multi-Protocol Permission Enforcement

(These are outlined in the design doc but deferred. Implement after Phase 1-6 are stable.)

---

## Summary

| Phase | Tasks | Effort | Key Deliverable |
|-------|-------|--------|-----------------|
| 1 | 9 tasks | Week 1 | New domain entities, parsers, DB migration |
| 2 | 5 tasks | Week 2-3 | Permission enforcement at runtime |
| 3 | 7 tasks | Week 3-4 | Convention-driven auto-CRUD from model |
| 4 | 6 tasks | Week 4-5 | Permission-aware bc-datatable with modal |
| 5 | 6 tasks | Week 5-7 | Full admin UI for groups + security sync |
| 6 | 8 tasks | Week 7-8 | CLI, Swagger, module updates, cleanup |
| 7 | 4 tasks | Week 8-10 | GraphQL + WebSocket (optional) |
| **Total** | **45 tasks** | **~8-10 weeks** | |
