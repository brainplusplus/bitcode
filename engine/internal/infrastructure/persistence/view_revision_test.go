package persistence

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := AutoMigrateViewRevisions(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestViewRevision_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViewRevisionRepository(db)

	rev, err := repo.Create("crm/contact_list", `{"name":"contact_list"}`, "admin")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if rev.Version != 1 {
		t.Errorf("expected version 1, got %d", rev.Version)
	}
	if rev.ViewKey != "crm/contact_list" {
		t.Errorf("expected crm/contact_list, got %s", rev.ViewKey)
	}

	rev2, err := repo.Create("crm/contact_list", `{"name":"contact_list","v":2}`, "admin")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if rev2.Version != 2 {
		t.Errorf("expected version 2, got %d", rev2.Version)
	}
}

func TestViewRevision_ListByViewKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViewRevisionRepository(db)

	repo.Create("crm/contact_list", `{"v":1}`, "admin")
	repo.Create("crm/contact_list", `{"v":2}`, "admin")
	repo.Create("crm/contact_list", `{"v":3}`, "admin")
	repo.Create("hrm/employee_list", `{"v":1}`, "admin")

	revs, err := repo.ListByViewKey("crm/contact_list", 0)
	if err != nil {
		t.Fatalf("ListByViewKey error: %v", err)
	}
	if len(revs) != 3 {
		t.Errorf("expected 3 revisions, got %d", len(revs))
	}
	if revs[0].Version != 3 {
		t.Errorf("expected latest first (version 3), got %d", revs[0].Version)
	}

	revs2, err := repo.ListByViewKey("crm/contact_list", 2)
	if err != nil {
		t.Fatalf("ListByViewKey error: %v", err)
	}
	if len(revs2) != 2 {
		t.Errorf("expected 2 revisions with limit, got %d", len(revs2))
	}
}

func TestViewRevision_GetByVersion(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViewRevisionRepository(db)

	repo.Create("crm/contact_list", `{"v":1}`, "admin")
	repo.Create("crm/contact_list", `{"v":2}`, "admin")

	rev, err := repo.GetByVersion("crm/contact_list", 1)
	if err != nil {
		t.Fatalf("GetByVersion error: %v", err)
	}
	if rev.Content != `{"v":1}` {
		t.Errorf("expected v1 content, got %s", rev.Content)
	}
}

func TestViewRevision_GetLatest(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViewRevisionRepository(db)

	repo.Create("crm/contact_list", `{"v":1}`, "admin")
	repo.Create("crm/contact_list", `{"v":2}`, "admin")

	rev, err := repo.GetLatest("crm/contact_list")
	if err != nil {
		t.Fatalf("GetLatest error: %v", err)
	}
	if rev.Version != 2 {
		t.Errorf("expected version 2, got %d", rev.Version)
	}
}

func TestViewRevision_Cleanup(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViewRevisionRepository(db)

	for i := 0; i < 5; i++ {
		repo.Create("crm/contact_list", `{}`, "admin")
	}

	if count := repo.Count("crm/contact_list"); count != 5 {
		t.Fatalf("expected 5 revisions, got %d", count)
	}

	if err := repo.Cleanup("crm/contact_list", 3); err != nil {
		t.Fatalf("Cleanup error: %v", err)
	}

	if count := repo.Count("crm/contact_list"); count != 3 {
		t.Errorf("expected 3 revisions after cleanup, got %d", count)
	}

	revs, _ := repo.ListByViewKey("crm/contact_list", 0)
	if revs[0].Version != 5 {
		t.Errorf("expected latest version 5 preserved, got %d", revs[0].Version)
	}
}

func TestViewRevision_Cleanup_NoOp(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViewRevisionRepository(db)

	repo.Create("crm/contact_list", `{}`, "admin")
	repo.Create("crm/contact_list", `{}`, "admin")

	if err := repo.Cleanup("crm/contact_list", 0); err != nil {
		t.Fatalf("Cleanup error: %v", err)
	}
	if count := repo.Count("crm/contact_list"); count != 2 {
		t.Errorf("expected 2 (no cleanup with keepN=0), got %d", count)
	}

	if err := repo.Cleanup("crm/contact_list", 10); err != nil {
		t.Fatalf("Cleanup error: %v", err)
	}
	if count := repo.Count("crm/contact_list"); count != 2 {
		t.Errorf("expected 2 (under limit), got %d", count)
	}
}
