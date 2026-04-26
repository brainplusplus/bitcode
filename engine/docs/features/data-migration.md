# Data Migration System

Laravel-style data migration/seeder system with multi-format support, custom processors, and batch tracking.

## Overview

The data migration system provides a structured way to seed data into your application. Each migration is a JSON file in the module's `migrations/` directory with a timestamped filename. Migrations are tracked in the `ir_migration` table and only run once.

## Migration File Format

```
migrations/
├── 20260101_000001_seed_tags.json
├── 20260101_000002_seed_contacts.json
└── 20260201_120000_import_leads.json
```

Filename format: `YYYYMMDD_HHMMSS_name.json`

## Migration JSON Schema

```json
{
  "name": "seed_contacts",
  "model": "contact",
  "module": "crm",
  "description": "Import initial contacts",
  "source": {
    "type": "json",
    "file": "data/contacts.json",
    "options": {
      "sheet": "Sheet1",
      "header_row": 1,
      "delimiter": ",",
      "root_element": "data.contacts",
      "skip_rows": 0
    }
  },
  "processor": {
    "type": "script",
    "script": { "lang": "typescript", "file": "scripts/transform.ts" }
  },
  "options": {
    "batch_size": 100,
    "on_conflict": "skip",
    "unique_fields": ["email"],
    "update_fields": ["name", "phone"],
    "generate_id": true,
    "hash_passwords": true,
    "set_timestamps": true,
    "dry_run": false
  },
  "field_mapping": {
    "contact_name": "name",
    "contact_email": "email"
  },
  "defaults": {
    "type": "person",
    "active": true
  },
  "down": {
    "strategy": "delete_by_source"
  }
}
```

## Source Types

| Type | Extension | Description |
|------|-----------|-------------|
| `json` | `.json` | JSON array or object with root_element path |
| `csv` | `.csv` | CSV with configurable delimiter and header row |
| `xlsx` | `.xlsx` | Excel with sheet name and header row config |
| `xml` | `.xml` | XML with root_element navigation |

### JSON Source

Supports both array format and object format:

```json
// Array format
[{"name": "Alice"}, {"name": "Bob"}]

// Object format (use root_element to navigate)
{"data": {"contacts": [{"name": "Alice"}]}}
// root_element: "data.contacts"
```

### CSV Source

```csv
name,email,age
Alice,alice@test.com,30
Bob,bob@test.com,25
```

Options: `delimiter` (default `,`), `header_row` (1-indexed), `skip_rows`.

### XLSX Source

Options: `sheet` (default: first sheet), `header_row` (1-indexed).

### XML Source

```xml
<contacts>
  <contact>
    <name>Alice</name>
    <email>alice@test.com</email>
  </contact>
</contacts>
```

Options: `root_element` for nested navigation (e.g., `data.contacts`).

## Conflict Modes

| Mode | Behavior |
|------|----------|
| `skip` | Skip records that already exist (default) |
| `upsert` | Update existing records, insert new ones (requires `unique_fields`) |
| `error` | Fail on duplicate |

## Processors

Custom data transformation before insertion.

### Script Processor

```json
{
  "processor": {
    "type": "script",
    "script": { "lang": "typescript", "file": "scripts/transform.ts" }
  }
}
```

The script receives `{ records, model, module }` and should return the transformed records array.

### Process Processor

```json
{
  "processor": {
    "type": "process",
    "process": "process_contact_import"
  }
}
```

The process receives `{ records, model, module }` in its input context.

## Rollback Strategies

| Strategy | Behavior |
|----------|----------|
| `none` | No rollback (default) |
| `delete_seeded` | Delete records by tracked IDs stored in ir_migration |
| `truncate` | Delete all records from the table |
| `custom` | Run a custom process or script |

## Module Configuration

Add `migrations` to your `module.json`:

```json
{
  "name": "crm",
  "migrations": ["migrations/*.json"],
  "data": ["data/demo.json"]
}
```

Migrations run automatically during module install, after the existing `data/*.json` seeder.

## CLI Commands

```bash
# Run pending migrations
bitcode seed run
bitcode seed run --module crm

# Rollback last batch
bitcode seed rollback
bitcode seed rollback --steps 2

# Show migration status
bitcode seed status
bitcode seed status --module crm

# Reset and re-run all
bitcode seed fresh
bitcode seed fresh --module crm

# Create new migration file
bitcode seed create seed_products --module crm --model product --type json
```

## Tracking Table (ir_migration)

| Column | Type | Description |
|--------|------|-------------|
| id | INT | Auto-increment primary key |
| name | VARCHAR(255) | Migration name |
| module | VARCHAR(100) | Module name |
| batch | INT | Batch number (for grouped rollback) |
| model | VARCHAR(255) | Target model |
| source | VARCHAR(50) | Source type (json/csv/xlsx/xml) |
| records | INT | Number of records processed |
| status | VARCHAR(20) | completed/failed |
| error | TEXT | Error message (if failed) |
| duration | BIGINT | Execution time in milliseconds |
| created_at | DATETIME | When the migration ran |

## Examples

### Direct JSON to model

```json
{
  "name": "seed_tags",
  "model": "tag",
  "source": { "type": "json", "file": "data/tags.json" }
}
```

### CSV with field mapping

```json
{
  "name": "import_contacts",
  "model": "contact",
  "source": { "type": "csv", "file": "data/contacts.csv" },
  "field_mapping": { "contact_name": "name", "contact_email": "email" },
  "defaults": { "type": "person" }
}
```

### XLSX with upsert

```json
{
  "name": "sync_products",
  "model": "product",
  "source": { "type": "xlsx", "file": "data/products.xlsx", "options": { "sheet": "Products" } },
  "options": {
    "on_conflict": "upsert",
    "unique_fields": ["sku"],
    "update_fields": ["name", "price", "stock"]
  }
}
```

### XML with custom processor

```json
{
  "name": "import_orders",
  "model": "order",
  "source": { "type": "xml", "file": "data/orders.xml", "options": { "root_element": "orders.order" } },
  "processor": { "type": "script", "script": { "lang": "typescript", "file": "scripts/transform_orders.ts" } }
}
```

## MongoDB Support

Full parity with SQL. The migration system uses a `MigrationStore` interface with both `GormMigrationStore` (SQL) and `MongoMigrationStore` (MongoDB) implementations. Data insertion uses a `DataInserter` interface with `GormDataInserter` and `MongoDataInserter`.

## Legacy Seeder Coordination

When a module has both `data` (legacy seeder) and `migrations` configured, the legacy seeder runs once and is tracked as `_legacy_seed` in ir_migration. Subsequent runs skip the legacy seeder. Modules without `migrations` continue using the legacy seeder as before.

## noupdate (Odoo-style)

```json
{
  "options": {
    "on_conflict": "upsert",
    "unique_fields": ["code"],
    "noupdate": true
  }
}
```

When `noupdate` is true, existing records are never overwritten. New records are inserted, but existing ones are skipped even in upsert mode. This preserves user customizations.

## field_types Override

```json
{
  "source": {
    "type": "csv",
    "file": "data/products.csv",
    "options": {
      "field_types": {
        "code": "string",
        "price": "float",
        "active": "boolean"
      }
    }
  }
}
```

Overrides automatic type inference for specific fields. Supported types: `string`, `int`, `float`, `bool`.

## Implementation

| File | Description |
|------|-------------|
| `compiler/parser/migration.go` | Migration JSON parser, file discovery, sorting |
| `infrastructure/persistence/migration_tracker.go` | MigrationStore interface, GormMigrationStore, MongoMigrationStore, record_ids tracking |
| `infrastructure/module/reader.go` | Multi-format data readers (JSON, CSV, XLSX, XML) |
| `infrastructure/module/migration.go` | MigrationEngine, DataInserter interface, GormDataInserter, MongoDataInserter, transaction wrapping |
| `infrastructure/module/migration_test.go` | 24 tests |
| `cmd/bitcode/seed.go` | CLI commands with dependency-ordered execution |
