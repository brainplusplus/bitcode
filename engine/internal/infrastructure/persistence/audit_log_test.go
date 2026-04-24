package persistence

import (
	"testing"
)

func setupAuditTestDB(t *testing.T) (*AuditLogRepository, func()) {
	db := setupDataRevisionTestDB(t)
	db.Exec(`CREATE TABLE IF NOT EXISTS audit_logs (
		id TEXT PRIMARY KEY,
		user_id TEXT,
		action TEXT,
		model_name TEXT,
		record_id TEXT,
		changes TEXT,
		ip_address TEXT,
		user_agent TEXT,
		request_method TEXT,
		request_path TEXT,
		status_code INTEGER,
		duration_ms INTEGER,
		created_at DATETIME,
		updated_at DATETIME,
		active INTEGER DEFAULT 1
	)`)
	repo := NewAuditLogRepository(db)
	return repo, func() {}
}

func TestAuditLogWrite(t *testing.T) {
	repo, cleanup := setupAuditTestDB(t)
	defer cleanup()

	err := repo.Write(AuditLogEntry{
		UserID:        "user-1",
		Action:        "login",
		ModelName:     "user",
		RecordID:      "user-1",
		IPAddress:     "127.0.0.1",
		UserAgent:     "Mozilla/5.0",
		RequestMethod: "POST",
		RequestPath:   "/auth/login",
		StatusCode:    200,
		DurationMs:    15,
	})
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	results, err := repo.FindLoginHistory(10)
	if err != nil {
		t.Fatalf("FindLoginHistory failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 login entry, got %d", len(results))
	}
}

func TestAuditLogFindByRecord(t *testing.T) {
	repo, cleanup := setupAuditTestDB(t)
	defer cleanup()

	repo.Write(AuditLogEntry{Action: "request", ModelName: "contact", RecordID: "abc-123", RequestMethod: "GET", RequestPath: "/api/contacts/abc-123"})
	repo.Write(AuditLogEntry{Action: "request", ModelName: "contact", RecordID: "abc-123", RequestMethod: "PUT", RequestPath: "/api/contacts/abc-123"})
	repo.Write(AuditLogEntry{Action: "request", ModelName: "order", RecordID: "xyz-456", RequestMethod: "GET", RequestPath: "/api/orders/xyz-456"})

	results, err := repo.FindByRecord("contact", "abc-123", 10)
	if err != nil {
		t.Fatalf("FindByRecord failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 entries for contact/abc-123, got %d", len(results))
	}
}

func TestAuditLogFindRequests(t *testing.T) {
	repo, cleanup := setupAuditTestDB(t)
	defer cleanup()

	repo.Write(AuditLogEntry{Action: "request", RequestMethod: "GET", RequestPath: "/api/contacts", StatusCode: 200})
	repo.Write(AuditLogEntry{Action: "request", RequestMethod: "POST", RequestPath: "/api/contacts", StatusCode: 201})
	repo.Write(AuditLogEntry{Action: "request", RequestMethod: "GET", RequestPath: "/api/orders", StatusCode: 200})
	repo.Write(AuditLogEntry{Action: "login", RequestMethod: "POST", RequestPath: "/auth/login"})

	all, _ := repo.FindRequests(10, "")
	if len(all) != 3 {
		t.Errorf("expected 3 request entries, got %d", len(all))
	}

	gets, _ := repo.FindRequests(10, "GET")
	if len(gets) != 2 {
		t.Errorf("expected 2 GET entries, got %d", len(gets))
	}

	posts, _ := repo.FindRequests(10, "POST")
	if len(posts) != 1 {
		t.Errorf("expected 1 POST entry, got %d", len(posts))
	}
}

func TestAuditLogFindByUser(t *testing.T) {
	repo, cleanup := setupAuditTestDB(t)
	defer cleanup()

	repo.Write(AuditLogEntry{UserID: "user-1", Action: "request", RequestPath: "/api/contacts"})
	repo.Write(AuditLogEntry{UserID: "user-1", Action: "login", RequestPath: "/auth/login"})
	repo.Write(AuditLogEntry{UserID: "user-2", Action: "request", RequestPath: "/api/orders"})

	results, err := repo.FindByUser("user-1", 10)
	if err != nil {
		t.Fatalf("FindByUser failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 entries for user-1, got %d", len(results))
	}
}

func TestAuditLogLoginHistory(t *testing.T) {
	repo, cleanup := setupAuditTestDB(t)
	defer cleanup()

	repo.Write(AuditLogEntry{UserID: "user-1", Action: "login"})
	repo.Write(AuditLogEntry{UserID: "user-1", Action: "logout"})
	repo.Write(AuditLogEntry{UserID: "user-2", Action: "register"})
	repo.Write(AuditLogEntry{UserID: "user-1", Action: "request"})

	results, err := repo.FindLoginHistory(10)
	if err != nil {
		t.Fatalf("FindLoginHistory failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 login/logout/register entries, got %d", len(results))
	}
}
