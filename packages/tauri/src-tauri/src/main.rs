#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri_plugin_sql::{Builder as SqlBuilder, Migration, MigrationKind};

fn offline_migrations() -> Vec<Migration> {
    vec![
        Migration {
            version: 1,
            description: "create_off_outbox",
            sql: "CREATE TABLE IF NOT EXISTS _off_outbox (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                envelope_id TEXT NOT NULL DEFAULT '',
                table_name TEXT NOT NULL,
                record_id TEXT NOT NULL,
                operation TEXT NOT NULL CHECK(operation IN ('CREATE','UPDATE','DELETE')),
                payload TEXT NOT NULL DEFAULT '{}',
                status TEXT NOT NULL DEFAULT 'PENDING' CHECK(status IN ('PENDING','SYNCED','ERROR','DEAD')),
                idempotency_key TEXT NOT NULL UNIQUE,
                device_id TEXT NOT NULL DEFAULT '',
                created_at TEXT NOT NULL DEFAULT (datetime('now')),
                retry_count INTEGER NOT NULL DEFAULT 0
            )",
            kind: MigrationKind::Up,
        },
        Migration {
            version: 2,
            description: "create_off_sync_state",
            sql: "CREATE TABLE IF NOT EXISTS _off_sync_state (
                device_id TEXT PRIMARY KEY,
                device_prefix TEXT NOT NULL DEFAULT '',
                last_sync_at TEXT NOT NULL DEFAULT '',
                last_pull_version INTEGER NOT NULL DEFAULT 0,
                registered_at TEXT NOT NULL DEFAULT '',
                auth_cached_at TEXT NOT NULL DEFAULT '',
                user_id TEXT NOT NULL DEFAULT '',
                user_hash TEXT NOT NULL DEFAULT '',
                failed_auth_attempts INTEGER NOT NULL DEFAULT 0,
                locked_until TEXT NOT NULL DEFAULT ''
            )",
            kind: MigrationKind::Up,
        },
        Migration {
            version: 3,
            description: "create_off_conflict_log",
            sql: "CREATE TABLE IF NOT EXISTS _off_conflict_log (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                table_name TEXT NOT NULL,
                record_id TEXT NOT NULL,
                field_name TEXT NOT NULL,
                local_value TEXT,
                remote_value TEXT,
                resolved_value TEXT,
                resolution TEXT NOT NULL DEFAULT 'LWW',
                resolved_at TEXT NOT NULL DEFAULT (datetime('now')),
                device_id TEXT NOT NULL DEFAULT ''
            )",
            kind: MigrationKind::Up,
        },
        Migration {
            version: 4,
            description: "create_off_number_sequence",
            sql: "CREATE TABLE IF NOT EXISTS _off_number_sequence (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                table_name TEXT NOT NULL,
                prefix TEXT NOT NULL DEFAULT '',
                last_sequence INTEGER NOT NULL DEFAULT 0,
                UNIQUE(table_name, prefix)
            )",
            kind: MigrationKind::Up,
        },
        Migration {
            version: 5,
            description: "create_off_auth_cache",
            sql: "CREATE TABLE IF NOT EXISTS _off_auth_cache (
                user_id TEXT PRIMARY KEY,
                user_hash TEXT NOT NULL,
                user_email TEXT NOT NULL DEFAULT '',
                user_name TEXT NOT NULL DEFAULT '',
                user_groups TEXT NOT NULL DEFAULT '[]',
                cached_at TEXT NOT NULL DEFAULT (datetime('now')),
                expires_at TEXT NOT NULL,
                device_id TEXT NOT NULL DEFAULT ''
            )",
            kind: MigrationKind::Up,
        },
    ]
}

fn db_connection_string() -> String {
    #[cfg(feature = "encryption")]
    {
        if let Ok(key) = std::env::var("BITCODE_DB_KEY") {
            return format!("sqlite:bitcode.db?key={}", key);
        }
    }
    "sqlite:bitcode.db".to_string()
}

fn main() {
    let mut builder = tauri::Builder::default();

    builder = builder
        .plugin(
            SqlBuilder::default()
                .add_migrations(&db_connection_string(), offline_migrations())
                .build(),
        )
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_notification::init());

    #[cfg(feature = "mobile-plugins")]
    {
        builder = builder
            .plugin(tauri_plugin_barcode_scanner::init())
            .plugin(tauri_plugin_biometric::init());
    }

    builder
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
