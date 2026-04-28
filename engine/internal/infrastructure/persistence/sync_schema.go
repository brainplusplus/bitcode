package persistence

import (
	"fmt"

	"gorm.io/gorm"
)

func CreateSyncInfrastructureTables(db *gorm.DB) error {
	dialect := DetectDialect(db)
	tables := []string{
		syncLogSQL(dialect),
		syncDevicesSQL(dialect),
		syncConflictsSQL(dialect),
		syncVersionsSQL(dialect),
	}
	for _, sql := range tables {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to create sync infrastructure table: %w", err)
		}
	}
	return nil
}

func syncLogSQL(dialect DBDialect) string {
	switch dialect {
	case DialectPostgres:
		return `CREATE TABLE IF NOT EXISTS _sync_log (
  envelope_id UUID PRIMARY KEY,
  device_id TEXT NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  status TEXT NOT NULL,
  operations_count INTEGER NOT NULL DEFAULT 0,
  response JSONB,
  error_message TEXT,
  processing_time_ms INTEGER
)`
	case DialectMySQL:
		return `CREATE TABLE IF NOT EXISTS _sync_log (
  envelope_id CHAR(36) PRIMARY KEY,
  device_id VARCHAR(255) NOT NULL,
  received_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  status VARCHAR(50) NOT NULL,
  operations_count INTEGER NOT NULL DEFAULT 0,
  response JSON,
  error_message TEXT,
  processing_time_ms INTEGER
)`
	default:
		return `CREATE TABLE IF NOT EXISTS _sync_log (
  envelope_id TEXT PRIMARY KEY,
  device_id TEXT NOT NULL,
  received_at TEXT NOT NULL DEFAULT (datetime('now')),
  status TEXT NOT NULL,
  operations_count INTEGER NOT NULL DEFAULT 0,
  response TEXT,
  error_message TEXT,
  processing_time_ms INTEGER
)`
	}
}

func syncDevicesSQL(dialect DBDialect) string {
	switch dialect {
	case DialectPostgres:
		return `CREATE TABLE IF NOT EXISTS _sync_devices (
  device_id TEXT PRIMARY KEY,
  device_prefix TEXT NOT NULL UNIQUE,
  device_name TEXT,
  platform TEXT NOT NULL,
  app_version TEXT,
  user_id UUID,
  tenant_id UUID,
  store_id UUID,
  registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_sync_at TIMESTAMPTZ,
  last_sync_version BIGINT NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deactivated_at TIMESTAMPTZ,
  deactivated_reason TEXT
)`
	case DialectMySQL:
		return `CREATE TABLE IF NOT EXISTS _sync_devices (
  device_id VARCHAR(255) PRIMARY KEY,
  device_prefix VARCHAR(50) NOT NULL UNIQUE,
  device_name VARCHAR(255),
  platform VARCHAR(50) NOT NULL,
  app_version VARCHAR(50),
  user_id CHAR(36),
  tenant_id CHAR(36),
  store_id CHAR(36),
  registered_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_sync_at DATETIME,
  last_sync_version BIGINT NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  deactivated_at DATETIME,
  deactivated_reason TEXT
)`
	default:
		return `CREATE TABLE IF NOT EXISTS _sync_devices (
  device_id TEXT PRIMARY KEY,
  device_prefix TEXT NOT NULL UNIQUE,
  device_name TEXT,
  platform TEXT NOT NULL,
  app_version TEXT,
  user_id TEXT,
  tenant_id TEXT,
  store_id TEXT,
  registered_at TEXT NOT NULL DEFAULT (datetime('now')),
  last_sync_at TEXT,
  last_sync_version INTEGER NOT NULL DEFAULT 0,
  is_active INTEGER NOT NULL DEFAULT 1,
  deactivated_at TEXT,
  deactivated_reason TEXT
)`
	}
}

func syncConflictsSQL(dialect DBDialect) string {
	switch dialect {
	case DialectPostgres:
		return `CREATE TABLE IF NOT EXISTS _sync_conflicts (
  id SERIAL PRIMARY KEY,
  envelope_id UUID NOT NULL,
  device_id TEXT NOT NULL,
  other_device_id TEXT,
  table_name TEXT NOT NULL,
  record_id UUID NOT NULL,
  field_name TEXT NOT NULL,
  device_value TEXT,
  server_value TEXT,
  resolved_value TEXT,
  resolution TEXT NOT NULL,
  auto_resolved BOOLEAN NOT NULL DEFAULT TRUE,
  reviewed_by UUID,
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  device_hlc TEXT,
  server_hlc TEXT
)`
	case DialectMySQL:
		return `CREATE TABLE IF NOT EXISTS _sync_conflicts (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  envelope_id CHAR(36) NOT NULL,
  device_id VARCHAR(255) NOT NULL,
  other_device_id VARCHAR(255),
  table_name VARCHAR(255) NOT NULL,
  record_id CHAR(36) NOT NULL,
  field_name VARCHAR(255) NOT NULL,
  device_value TEXT,
  server_value TEXT,
  resolved_value TEXT,
  resolution VARCHAR(50) NOT NULL,
  auto_resolved BOOLEAN NOT NULL DEFAULT TRUE,
  reviewed_by CHAR(36),
  reviewed_at DATETIME,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  device_hlc VARCHAR(255),
  server_hlc VARCHAR(255)
)`
	default:
		return `CREATE TABLE IF NOT EXISTS _sync_conflicts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  envelope_id TEXT NOT NULL,
  device_id TEXT NOT NULL,
  other_device_id TEXT,
  table_name TEXT NOT NULL,
  record_id TEXT NOT NULL,
  field_name TEXT NOT NULL,
  device_value TEXT,
  server_value TEXT,
  resolved_value TEXT,
  resolution TEXT NOT NULL,
  auto_resolved INTEGER NOT NULL DEFAULT 1,
  reviewed_by TEXT,
  reviewed_at TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  device_hlc TEXT,
  server_hlc TEXT
)`
	}
}

func syncVersionsSQL(dialect DBDialect) string {
	switch dialect {
	case DialectPostgres:
		return `CREATE TABLE IF NOT EXISTS _sync_versions (
  id SERIAL PRIMARY KEY,
  table_name TEXT NOT NULL,
  record_id UUID NOT NULL,
  operation TEXT NOT NULL,
  version BIGSERIAL NOT NULL,
  changed_fields JSONB,
  changed_by TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sync_versions_version ON _sync_versions(version);
CREATE INDEX IF NOT EXISTS idx_sync_versions_record ON _sync_versions(table_name, record_id)`
	case DialectMySQL:
		return `CREATE TABLE IF NOT EXISTS _sync_versions (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  table_name VARCHAR(255) NOT NULL,
  record_id CHAR(36) NOT NULL,
  operation VARCHAR(50) NOT NULL,
  version BIGINT NOT NULL,
  changed_fields JSON,
  changed_by VARCHAR(255),
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_sync_versions_version (version),
  INDEX idx_sync_versions_record (table_name, record_id)
)`
	default:
		return `CREATE TABLE IF NOT EXISTS _sync_versions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  table_name TEXT NOT NULL,
  record_id TEXT NOT NULL,
  operation TEXT NOT NULL,
  version INTEGER NOT NULL,
  changed_fields TEXT,
  changed_by TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`
	}
}
