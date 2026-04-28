package persistence

import (
	"fmt"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"gorm.io/gorm"
)

func OfflineColumns(dialect DBDialect) string {
	if dialect != DialectSQLite {
		return ""
	}
	return `,
  _off_device_id TEXT NOT NULL DEFAULT '',
  _off_status INTEGER NOT NULL DEFAULT 0,
  _off_version INTEGER NOT NULL DEFAULT 1,
  _off_deleted INTEGER NOT NULL DEFAULT 0,
  _off_created_at TEXT NOT NULL DEFAULT '',
  _off_updated_at TEXT NOT NULL DEFAULT '',
  _off_hlc TEXT NOT NULL DEFAULT '',
  _off_envelope_id TEXT`
}

func OfflineUUIDColumn(pk *parser.PrimaryKeyConfig, dialect DBDialect) string {
	if dialect != DialectSQLite {
		return ""
	}
	if pk == nil || pk.Strategy == parser.PKUUID {
		return ""
	}
	return ",\n  _off_uuid TEXT NOT NULL UNIQUE"
}

func CreateOfflineInfrastructureTables(db *gorm.DB) error {
	tables := []string{
		offlineOutboxSQL(),
		offlineSyncStateSQL(),
		offlineConflictLogSQL(),
		offlineNumberSequenceSQL(),
	}
	for _, sql := range tables {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to create offline infrastructure table: %w", err)
		}
	}
	return nil
}

func offlineOutboxSQL() string {
	return `CREATE TABLE IF NOT EXISTS _off_outbox (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  envelope_id TEXT NOT NULL,
  table_name TEXT NOT NULL,
  record_id TEXT NOT NULL,
  operation TEXT NOT NULL,
  payload TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'PENDING',
  idempotency_key TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL,
  retry_count INTEGER NOT NULL DEFAULT 0
)`
}

func offlineSyncStateSQL() string {
	return `CREATE TABLE IF NOT EXISTS _off_sync_state (
  device_id TEXT PRIMARY KEY,
  device_prefix TEXT NOT NULL,
  last_sync_at TEXT,
  last_pull_version INTEGER NOT NULL DEFAULT 0,
  registered_at TEXT NOT NULL,
  auth_cached_at TEXT,
  user_id TEXT,
  user_hash TEXT
)`
}

func offlineConflictLogSQL() string {
	return `CREATE TABLE IF NOT EXISTS _off_conflict_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  table_name TEXT NOT NULL,
  record_id TEXT NOT NULL,
  field_name TEXT NOT NULL,
  local_value TEXT,
  remote_value TEXT,
  resolved_value TEXT,
  resolution TEXT NOT NULL,
  resolved_at TEXT NOT NULL,
  device_id TEXT NOT NULL
)`
}

func offlineNumberSequenceSQL() string {
	return `CREATE TABLE IF NOT EXISTS _off_number_sequence (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  table_name TEXT NOT NULL,
  prefix TEXT NOT NULL,
  last_sequence INTEGER NOT NULL DEFAULT 0,
  UNIQUE(table_name, prefix)
)`
}
