package persistence

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupDataRevisionTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := AutoMigrateDataRevisions(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestDataRevisionCreate(t *testing.T) {
	db := setupDataRevisionTestDB(t)
	repo := NewDataRevisionRepository(db)

	snapshot := map[string]any{"name": "Test", "amount": 100}
	rev, err := repo.Create("contact", "abc-123", "create", snapshot, nil, "user-1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if rev.Version != 1 {
		t.Errorf("expected version 1, got %d", rev.Version)
	}
	if rev.Action != "create" {
		t.Errorf("expected action 'create', got %s", rev.Action)
	}
	if rev.ModelName != "contact" {
		t.Errorf("expected model 'contact', got %s", rev.ModelName)
	}
}

func TestDataRevisionVersionIncrement(t *testing.T) {
	db := setupDataRevisionTestDB(t)
	repo := NewDataRevisionRepository(db)

	snapshot1 := map[string]any{"name": "V1"}
	rev1, _ := repo.Create("contact", "abc-123", "create", snapshot1, nil, "user-1")

	snapshot2 := map[string]any{"name": "V2"}
	changes := map[string]any{"name": map[string]any{"old": "V1", "new": "V2"}}
	rev2, _ := repo.Create("contact", "abc-123", "update", snapshot2, changes, "user-1")

	if rev1.Version != 1 {
		t.Errorf("expected version 1, got %d", rev1.Version)
	}
	if rev2.Version != 2 {
		t.Errorf("expected version 2, got %d", rev2.Version)
	}
}

func TestDataRevisionListByRecord(t *testing.T) {
	db := setupDataRevisionTestDB(t)
	repo := NewDataRevisionRepository(db)

	repo.Create("contact", "abc-123", "create", map[string]any{"v": 1}, nil, "user-1")
	repo.Create("contact", "abc-123", "update", map[string]any{"v": 2}, nil, "user-1")
	repo.Create("contact", "abc-123", "update", map[string]any{"v": 3}, nil, "user-1")
	repo.Create("contact", "other-id", "create", map[string]any{"v": 1}, nil, "user-1")

	revisions, err := repo.ListByRecord("contact", "abc-123", 50)
	if err != nil {
		t.Fatalf("ListByRecord failed: %v", err)
	}
	if len(revisions) != 3 {
		t.Errorf("expected 3 revisions, got %d", len(revisions))
	}
	if revisions[0].Version != 3 {
		t.Errorf("expected latest version 3 first, got %d", revisions[0].Version)
	}
}

func TestDataRevisionGetByVersion(t *testing.T) {
	db := setupDataRevisionTestDB(t)
	repo := NewDataRevisionRepository(db)

	repo.Create("contact", "abc-123", "create", map[string]any{"name": "Original"}, nil, "user-1")
	repo.Create("contact", "abc-123", "update", map[string]any{"name": "Updated"}, nil, "user-1")

	rev, err := repo.GetByVersion("contact", "abc-123", 1)
	if err != nil {
		t.Fatalf("GetByVersion failed: %v", err)
	}

	snapshot, err := repo.GetSnapshotMap(rev)
	if err != nil {
		t.Fatalf("GetSnapshotMap failed: %v", err)
	}
	if snapshot["name"] != "Original" {
		t.Errorf("expected 'Original', got %v", snapshot["name"])
	}
}

func TestDataRevisionCleanup(t *testing.T) {
	db := setupDataRevisionTestDB(t)
	repo := NewDataRevisionRepository(db)

	for i := 0; i < 10; i++ {
		repo.Create("contact", "abc-123", "update", map[string]any{"v": i}, nil, "user-1")
	}

	count := repo.Count("contact", "abc-123")
	if count != 10 {
		t.Errorf("expected 10 revisions, got %d", count)
	}

	repo.Cleanup("contact", "abc-123", 5)

	count = repo.Count("contact", "abc-123")
	if count != 5 {
		t.Errorf("expected 5 revisions after cleanup, got %d", count)
	}
}

func TestComputeChanges(t *testing.T) {
	before := map[string]any{
		"name":   "Old Name",
		"amount": 100,
		"status": "draft",
	}
	after := map[string]any{
		"name":   "New Name",
		"amount": 100,
		"status": "active",
	}

	changes := ComputeChanges(before, after)

	if _, ok := changes["amount"]; ok {
		t.Error("amount should not be in changes (unchanged)")
	}

	nameChange, ok := changes["name"].(map[string]any)
	if !ok {
		t.Fatal("name should be in changes")
	}
	if nameChange["old"] != "Old Name" || nameChange["new"] != "New Name" {
		t.Errorf("unexpected name change: %v", nameChange)
	}

	statusChange, ok := changes["status"].(map[string]any)
	if !ok {
		t.Fatal("status should be in changes")
	}
	if statusChange["old"] != "draft" || statusChange["new"] != "active" {
		t.Errorf("unexpected status change: %v", statusChange)
	}
}

func TestDataRevisionGetLatest(t *testing.T) {
	db := setupDataRevisionTestDB(t)
	repo := NewDataRevisionRepository(db)

	repo.Create("contact", "abc-123", "create", map[string]any{"v": 1}, nil, "user-1")
	repo.Create("contact", "abc-123", "update", map[string]any{"v": 2}, nil, "user-1")

	latest, err := repo.GetLatest("contact", "abc-123")
	if err != nil {
		t.Fatalf("GetLatest failed: %v", err)
	}
	if latest.Version != 2 {
		t.Errorf("expected version 2, got %d", latest.Version)
	}
}
