# MongoDB Support & Unified Query Builder — Design

**Date**: 2026-04-25
**Status**: Approved
**Scope**: Phase 2 — Full parity MongoDB support, unified query builder, built-in model functions

## Overview

Abstract the entire persistence layer behind a `Repository` interface. Two implementations: `GormRepository` (SQL — SQLite/Postgres/MySQL) and `MongoRepository` (MongoDB). Unified query builder translates to both backends. Built-in model functions callable from processes and plugin scripts.

## Design Decisions

1. **Repository interface** — all consumers use interface, not concrete type
2. **Extended Reference Pattern** for MongoDB many2one — embed `title_field` value in `_refs` object
3. **No auto-sync** for `_refs` — populated on write, stale acceptable for list views
4. **Many2many** — SQL uses junction tables, MongoDB uses arrays of IDs + `_refs`
5. **Separate MongoDB sequence engine** — counter collection pattern
6. **Search** uses `$or` + `$regex` (not text index) for flexibility
7. **MongoDB transactions** for multi-document operations (requires replica set)
8. **`title_field`** and **`search_field`** added to model JSON — cross-cutting, benefits both backends

## Model JSON — New Fields

```json
{
  "name": "contact",
  "title_field": "name",
  "search_field": ["name", "email", "company"],
  "fields": { ... }
}
```

### title_field

Determines which field is the "display name" for this model. Used in:
- Extended references (MongoDB `_refs._title`)
- Link display in views
- Admin panel record labels

Default resolution chain: `name` → `label` → `title` → `code` → `username` → `description` → first string/text field excluding id.

Not pre-computed — resolved at runtime from the record's actual field value.

### search_field

Array of field names to search when `?q=term` is used. Default: `[title_field]`.

SQL: `WHERE field1 LIKE '%term%' OR field2 LIKE '%term%'`
MongoDB: `{$or: [{field1: {$regex: "term", $options: "i"}}, ...]}`

## Repository Interface

```go
type Repository interface {
    // Core CRUD
    FindByID(ctx context.Context, id string) (map[string]any, error)
    FindAll(ctx context.Context, query *Query, page, pageSize int) ([]map[string]any, int64, error)
    Create(ctx context.Context, data map[string]any) (map[string]any, error)
    Update(ctx context.Context, id string, data map[string]any) error
    Delete(ctx context.Context, id string) error
    SoftDelete(ctx context.Context, id string) error

    // Extended operations
    Upsert(ctx context.Context, data map[string]any, uniqueFields []string) (map[string]any, error)
    Count(ctx context.Context, query *Query) (int64, error)
    Sum(ctx context.Context, field string, query *Query) (float64, error)
    BulkCreate(ctx context.Context, records []map[string]any) ([]map[string]any, error)

    // Relations
    AddMany2Many(ctx context.Context, id string, field string, relatedIDs []string) error
    RemoveMany2Many(ctx context.Context, id string, field string, relatedIDs []string) error
    LoadMany2Many(ctx context.Context, id string, field string) ([]map[string]any, error)

    // Metadata
    SetHydrator(h HydratorInterface)
    SetRevisionRepo(r RevisionRepository)
    SetModelName(name string)
    SetEncryptor(enc EncryptorInterface)
    SetModelDef(def *parser.ModelDefinition)
    TableName() string
}
```

## Query Builder

```go
type Query struct {
    Conditions []Condition
    OrGroups   [][]Condition  // OR groups: each inner slice is ANDed, groups are ORed
    OrderBy    []OrderClause
    Limit      int
    Offset     int
    Select     []string
    GroupBy    []string
}

type Condition struct {
    Field    string
    Operator string  // =, !=, >, <, >=, <=, like, in, not_in, between, is_null, is_not_null
    Value    any
}

type OrderClause struct {
    Field     string
    Direction string // "asc", "desc"
}
```

### JSON DSL (for plugins/processes)

```json
{
  "wheres": [
    {"field": "status", "op": "=", "value": "active"},
    {"field": "revenue", "op": ">", "value": 1000}
  ],
  "order": [{"field": "name", "direction": "asc"}],
  "limit": 20,
  "offset": 0
}
```

### Translation

**SQL (GORM)**:
```
Condition{Field: "status", Operator: "=", Value: "active"}
→ db.Where("status = ?", "active")

Condition{Field: "name", Operator: "like", Value: "%Tech%"}
→ db.Where("name LIKE ?", "%Tech%")

Condition{Field: "id", Operator: "in", Value: ["a","b","c"]}
→ db.Where("id IN ?", ["a","b","c"])
```

**MongoDB**:
```
Condition{Field: "status", Operator: "=", Value: "active"}
→ bson.M{"status": "active"}

Condition{Field: "name", Operator: "like", Value: "%Tech%"}
→ bson.M{"name": bson.M{"$regex": "Tech", "$options": "i"}}

Condition{Field: "id", Operator: "in", Value: ["a","b","c"]}
→ bson.M{"_id": bson.M{"$in": ["a","b","c"]}}
```

## SQL Backend (GormRepository)

Refactored from existing `GenericRepository`. Same logic, implements `Repository` interface.

- many2many: junction tables (existing)
- Sequence: existing `SequenceEngine` (renamed to `GormSequenceEngine`)
- Audit/Revisions: refactored to use Repository interface

New methods: `Upsert`, `Count`, `Sum`, `BulkCreate`, `AddMany2Many`, `RemoveMany2Many`, `LoadMany2Many`.

## MongoDB Backend (MongoRepository)

### Connection

```toml
[database]
driver = "mongodb"
host = "localhost"
port = 27017
name = "bitcode"
user = ""
password = ""
```

Env: `DB_DRIVER=mongodb DB_HOST=localhost DB_PORT=27017 DB_NAME=bitcode`

### Collection Naming

Uses `TableName()` resolver from Phase 1. Model `contact` in CRM module → collection `crm_contact`.

### Extended Reference Pattern (many2one)

On Create/Update, when a many2one field is set:

```json
{
  "_id": "uuid-456",
  "name": "Big Deal",
  "contact_id": "uuid-123",
  "_refs": {
    "contact_id": {
      "_id": "uuid-123",
      "_title": "Budi Santoso"
    }
  }
}
```

- `_refs` auto-populated by looking up the referenced document's `title_field`
- No auto-sync — `_refs` populated on write only
- `_id` in `_refs` is always correct; `_title` may be stale

### Many2Many (MongoDB)

```json
{
  "_id": "uuid-123",
  "name": "Budi",
  "tag_ids": ["tag-1", "tag-2"],
  "_refs": {
    "tag_ids": [
      {"_id": "tag-1", "_title": "VIP"},
      {"_id": "tag-2", "_title": "Partner"}
    ]
  }
}
```

- `AddMany2Many` → `$addToSet` on array field + update `_refs`
- `RemoveMany2Many` → `$pull` from array + update `_refs`
- `LoadMany2Many` → `$in` query on related collection

### Migration (MongoDB)

- Collections auto-created on first insert
- Indexes created explicitly:
  - Unique indexes for unique fields
  - Compound indexes for frequently queried field combinations
  - No text indexes (search uses `$regex`)
- Optional JSON Schema validation (not enforced by default)

### Sequence Engine (MongoDB)

Separate file: `mongo_sequence.go`

Counter collection: `_sequences`

```go
// findOneAndUpdate with $inc, upsert: true
result := db.Collection("_sequences").FindOneAndUpdate(
    ctx,
    bson.M{"model": modelName, "field": fieldName, "key": sequenceKey},
    bson.M{"$inc": bson.M{"next_value": step}},
    options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
)
```

### Transactions

Multi-document operations (Create with many2many, cascading Delete) use MongoDB transactions.
Requires replica set. For standalone MongoDB, operations are best-effort (documented).

## System Repositories (Audit, Revisions)

`AuditLogRepository`, `ViewRevisionRepository`, `DataRevisionRepository` refactored to use `Repository` interface internally, or implement a simpler `SystemRepository` interface:

```go
type SystemRepository interface {
    Insert(ctx context.Context, collection string, data map[string]any) error
    Find(ctx context.Context, collection string, query *Query, page, pageSize int) ([]map[string]any, int64, error)
    FindOne(ctx context.Context, collection string, query *Query) (map[string]any, error)
}
```

This avoids the full Repository interface overhead for simple system tables.

## Seeder

Refactored to use `Repository` interface instead of raw GORM. `SeedModule` receives a repository factory function.

## Admin Panel

`loadUserRoles` / `loadUserGroups` refactored from raw SQL JOIN to repository-based queries:
1. Query junction table/array for user's role IDs
2. Query role collection for role names by IDs

## Built-in Model Functions

### Process Steps (extended)

New step types added to process engine:

```json
{"type": "upsert", "model": "contact", "set": {"email": "x@y.com"}, "unique": ["email"]}
{"type": "count", "model": "contact", "domain": [["status", "=", "active"]], "into": "total"}
{"type": "sum", "model": "contact", "field": "revenue", "domain": [], "into": "total_revenue"}
```

### Model Process Registry

Every registered model auto-gets process functions:

```
models.{model_name}.Get        → FindByID
models.{model_name}.Find       → FindByID (alias)
models.{model_name}.GetAll     → FindAll
models.{model_name}.FindAll    → FindAll (alias)
models.{model_name}.Paginate   → FindAll with pagination
models.{model_name}.Create     → Create
models.{model_name}.Update     → Update
models.{model_name}.Delete     → Delete
models.{model_name}.Upsert     → Upsert
models.{model_name}.Count      → Count
models.{model_name}.Sum        → Sum
```

Callable from:
- Process steps via `{"type": "call", "process": "models.contact.Count", ...}`
- TypeScript plugins via `Process("models.contact.Get", {...})`
- Python plugins via `Process("models.contact.Get", {...})`

## File Structure

```
engine/internal/infrastructure/persistence/
├── repository.go           # Repository interface + Query types + factory
├── query.go                # Query builder + JSON DSL parser
├── gorm_repository.go      # SQL backend (refactored from generic_repository.go)
├── gorm_migration.go       # SQL migration (refactored from dynamic_model.go)
├── gorm_sequence.go        # SQL sequence (renamed from sequence.go)
├── gorm_system.go          # SQL system repos (audit, revisions)
├── mongo_connection.go     # MongoDB connection setup
├── mongo_repository.go     # MongoDB backend
├── mongo_migration.go      # MongoDB migration (indexes)
├── mongo_sequence.go       # MongoDB sequence (counter collection)
├── mongo_system.go         # MongoDB system repos (audit, revisions)
├── database.go             # Existing SQL connections + MongoDB connection
├── audit_log.go            # Refactored to use SystemRepository
├── view_revision.go        # Refactored to use SystemRepository
├── data_revision.go        # Refactored to use SystemRepository
└── ...

engine/internal/runtime/
├── model_registry.go       # Model process registry (models.{name}.{op})
└── ...
```

## Config

```toml
[database]
driver = "mongodb"    # "sqlite" | "postgres" | "mysql" | "mongodb"
host = "localhost"
port = 27017
name = "bitcode"
user = ""
password = ""
```

`DB_DRIVER=mongodb` switches entire app to MongoDB backend.

## Breaking Changes

- `*GenericRepository` → `Repository` interface in all consumers
- `NewGenericRepository()` → `NewRepository()` factory
- New dependency: `go.mongodb.org/mongo-driver/v2`
- New model JSON fields: `title_field`, `search_field` (optional, backward compatible)
- New process step types: `upsert`, `count`, `sum`
- New model process registry: `models.{name}.{op}`

## Testing

- Unit tests for Query builder (both SQL and MongoDB translation)
- Unit tests for GormRepository (existing tests adapted)
- Integration tests for MongoRepository (requires MongoDB instance or mock)
- Unit tests for model process registry
- Full test suite: `go test ./...`
