# Phase 6B: Polymorphic Relations (Morphs)

**Date**: 14 July 2026
**Status**: Draft
**Depends on**: Phase 6A (new field types, storage hints)
**Unlocks**: Phase 6C (engine enhancements), Phase 7 (module setting)
**Master doc**: `2026-07-14-runtime-engine-redesign-master.md`

---

## Table of Contents

1. [Goal](#1-goal)
2. [What Are Polymorphic Relations](#2-what-are-polymorphic-relations)
3. [Current State: What Exists](#3-current-state-what-exists)
4. [Design: Four Morph Types](#4-design-four-morph-types)
5. [JSON Schema for Morphs](#5-json-schema-for-morphs)
6. [Database Schema](#6-database-schema)
7. [Parser Changes](#7-parser-changes)
8. [Migration: Column & Index Generation](#8-migration-column--index-generation)
9. [Repository: Loading Morph Relations](#9-repository-loading-morph-relations)
10. [Bridge API: Morph Operations](#10-bridge-api-morph-operations)
11. [View Layer: Morph Fields in UI](#11-view-layer-morph-fields-in-ui)
12. [GraphQL: Morph Types](#12-graphql-morph-types)
13. [API: REST Endpoints for Morphs](#13-api-rest-endpoints-for-morphs)
14. [Morph Map: Type Aliasing](#14-morph-map-type-aliasing)
15. [MongoDB Support](#15-mongodb-support)
16. [Edge Cases & Constraints](#16-edge-cases--constraints)
17. [Relationship with `dynamic_link`](#17-relationship-with-dynamic_link)
18. [Implementation Tasks](#18-implementation-tasks)

---

## 1. Goal

Add polymorphic relations (morphs) to the engine — allowing a model to belong to or have multiple different model types through a single relation. This is one of the most requested features and is critical for real-world applications.

### 1.1 Use Cases

| Use Case | Without Morphs | With Morphs |
|----------|---------------|-------------|
| Comments on posts AND videos | Two tables: `post_comments`, `video_comments` | One table: `comments` with `commentable_type` + `commentable_id` |
| Images for users AND products | Two FK columns: `user_id`, `product_id` (mostly NULL) | One pair: `imageable_type` + `imageable_id` |
| Tags on posts AND videos AND products | Three junction tables | One junction table: `taggables` |
| Activity log for any model | One FK per model (explosion of columns) | `subject_type` + `subject_id` |
| Attachments for invoices AND expenses | Duplicate attachment logic | Single `attachable` morph |

### 1.2 Success Criteria

- Four morph types work: `morph_to`, `morph_one`, `morph_many`, `morph_to_many`
- Morph columns auto-created during migration (`{name}_type` VARCHAR + `{name}_id` UUID/INT)
- Morph relations loadable via `with` clause (eager loading)
- Morph relations accessible via bridge API (`bitcode.db`)
- Morph relations renderable in views (form, list)
- Morph map for type aliasing (short names instead of full model paths)
- Works on all 4 databases (PostgreSQL, MySQL, SQLite, MongoDB)

### 1.3 What This Phase Does NOT Do

- Does not add morph-based permissions/record rules (future enhancement)
- Does not add morph-based cascade delete (manual via events for now)
- Does not change existing relation types (many2one, one2many, many2many)

---

## 2. What Are Polymorphic Relations

### 2.1 The Problem

Normal relations are **fixed** — a `comment` belongs to a `post`:

```
comments table:
  id | body          | post_id
  1  | "Great post!" | 42
```

But what if comments can belong to posts AND videos AND photos? Without morphs:

```
comments table (BAD — column explosion):
  id | body          | post_id | video_id | photo_id
  1  | "Great post!" | 42      | NULL     | NULL
  2  | "Nice video!" | NULL    | 7        | NULL
```

Every new commentable model = new nullable FK column. This doesn't scale.

### 2.2 The Solution: Polymorphic Columns

```
comments table (GOOD — polymorphic):
  id | body          | commentable_type | commentable_id
  1  | "Great post!" | "post"           | 42
  2  | "Nice video!" | "video"          | 7
  3  | "Cool photo!" | "photo"          | 15
```

Two columns replace N columns:
- `commentable_type` — which model this comment belongs to
- `commentable_id` — the ID of that model's record

### 2.3 Four Types of Polymorphic Relations

| Type | Laravel | Description | Example |
|------|---------|-------------|---------|
| **morph_to** | `morphTo()` | Child belongs to any parent type | Comment → Post OR Video |
| **morph_one** | `morphOne()` | Parent has one polymorphic child | User → one Image |
| **morph_many** | `morphMany()` | Parent has many polymorphic children | Post → many Comments |
| **morph_to_many** | `morphToMany()` | Many-to-many polymorphic (parent side) | Post → many Tags |
| **morph_by_many** | `morphedByMany()` | Many-to-many polymorphic (inverse/shared side) | Tag → many Posts, Videos |

---

## 3. Current State: What Exists

### 3.1 Existing Relation Types

| Type | Status | Storage |
|------|--------|---------|
| `many2one` | ✅ Full | FK column (`{field}_id`) |
| `one2many` | ✅ Full | Virtual (inverse of many2one) |
| `many2many` | ✅ Full | Junction table (`{model}_{field}`) |
| `dynamic_link` | ⚠️ Partial | VARCHAR column, requires `model` specified |

### 3.2 `dynamic_link` — Almost Polymorphic, But Not

`dynamic_link` currently requires a **fixed** `model` in the field definition:

```json
{ "ref": { "type": "dynamic_link", "model": "contact" } }
```

This is just a many2one with a different UI widget. It's NOT polymorphic because the target model is fixed at definition time.

**True polymorphic** means the target model is determined **per record** at runtime.

### 3.3 What Needs to Change

| Layer | Current | Needed for Morphs |
|-------|---------|-------------------|
| Parser | No morph field types | 5 new FieldTypes |
| Migration | No morph column generation | Auto-create `_type` + `_id` columns |
| Repository | No morph loading | `loadMorphToRelation`, `loadMorphManyRelation`, etc. |
| Bridge API | No morph operations | `bitcode.db.morphTo()`, `bitcode.db.morphMany()` |
| View | No morph rendering | Dynamic model selector + record selector |
| GraphQL | No morph types | Union types or interface types |
| API | No morph endpoints | Include morph data in responses |

---

## 4. Design: Four Morph Types

### 4.1 `morph_to` — "I belong to something, but I don't know what type until runtime"

**Who defines it**: The child model (the one with the `_type` + `_id` columns).

```json
// comments model
{
  "name": "comment",
  "fields": {
    "body": { "type": "text" },
    "commentable": {
      "type": "morph_to"
    }
  }
}
```

**Database columns created**: `commentable_type` (VARCHAR) + `commentable_id` (UUID/TEXT)

**No `model` needed** — the whole point is that ANY model can be the parent.

### 4.2 `morph_one` — "I have exactly one of this child, and other models might too"

**Who defines it**: The parent model.

```json
// user model
{
  "name": "user",
  "fields": {
    "avatar": {
      "type": "morph_one",
      "model": "image",
      "morph": "imageable"
    }
  }
}
```

**No database columns** — this is virtual (like `one2many`). The columns are on the `image` table (`imageable_type` + `imageable_id`).

`morph` = the name of the `morph_to` field on the child model.

### 4.3 `morph_many` — "I have many of these children, and other models might too"

**Who defines it**: The parent model.

```json
// post model
{
  "name": "post",
  "fields": {
    "comments": {
      "type": "morph_many",
      "model": "comment",
      "morph": "commentable"
    }
  }
}
```

**No database columns** — virtual. The columns are on the `comment` table.

### 4.4 `morph_to_many` — "I have many of these through a polymorphic junction table"

**Who defines it**: The parent model (Post, Video — the models being tagged).

```json
// post model
{
  "name": "post",
  "fields": {
    "tags": {
      "type": "morph_to_many",
      "model": "tag",
      "morph": "taggable"
    }
  }
}
```

**Junction table created**: `taggables` with columns:
- `id` (PK)
- `tag_id` (FK to tags)
- `taggable_id` (FK to parent)
- `taggable_type` (VARCHAR — "post", "video", etc.)

**Junction table naming**: Always pluralized morph name (`taggable` → `taggables`), regardless of project `table_naming` config. This is a convention — morph pivot tables are relation tables, not entity tables.

### 4.5 `morph_by_many` — "I am referenced by many different models through a polymorphic junction"

**Who defines it**: The shared model (Tag — the model being attached to multiple parent types).

```json
// tag model
{
  "name": "tag",
  "fields": {
    "posts": {
      "type": "morph_by_many",
      "model": "post",
      "morph": "taggable"
    },
    "videos": {
      "type": "morph_by_many",
      "model": "video",
      "morph": "taggable"
    }
  }
}
```

**No junction table created** — `morph_by_many` uses the same junction table created by `morph_to_many`. It is the inverse/read side of the relationship.

**Why a separate type instead of `inverse: true`?**
- More explicit — developer reads `morph_by_many` and immediately knows "I am the shared side"
- Familiar to Laravel developers (`morphedByMany`)
- Cleaner for codegen, introspection, and schema tooling
- No ambiguity in large schemas

---

## 5. JSON Schema for Morphs

### 5.1 `morph_to` (child side — creates columns)

```json
{
  "field_name": {
    "type": "morph_to",
    "models": ["post", "video", "photo"],
    "required": false
  }
}
```

| Property | Type | Required | Description |
|----------|------|:--------:|-------------|
| `type` | `"morph_to"` | ✅ | Polymorphic belongs-to |
| `models` | `string[]` | ❌ | Allowed parent models. If omitted = any model. If specified = validation + UI dropdown filter. |
| `required` | `bool` | ❌ | Whether the morph relation is required |

### 5.2 `morph_one` (parent side — virtual)

```json
{
  "field_name": {
    "type": "morph_one",
    "model": "image",
    "morph": "imageable"
  }
}
```

| Property | Type | Required | Description |
|----------|------|:--------:|-------------|
| `type` | `"morph_one"` | ✅ | One polymorphic child |
| `model` | `string` | ✅ | Target child model |
| `morph` | `string` | ✅ | Name of the `morph_to` field on the child model |

### 5.3 `morph_many` (parent side — virtual)

```json
{
  "field_name": {
    "type": "morph_many",
    "model": "comment",
    "morph": "commentable"
  }
}
```

| Property | Type | Required | Description |
|----------|------|:--------:|-------------|
| `type` | `"morph_many"` | ✅ | Many polymorphic children |
| `model` | `string` | ✅ | Target child model |
| `morph` | `string` | ✅ | Name of the `morph_to` field on the child model |

### 5.4 `morph_to_many` (parent side — creates junction table)

```json
{
  "field_name": {
    "type": "morph_to_many",
    "model": "tag",
    "morph": "taggable"
  }
}
```

| Property | Type | Required | Description |
|----------|------|:--------:|-------------|
| `type` | `"morph_to_many"` | ✅ | Many-to-many polymorphic (parent side) |
| `model` | `string` | ✅ | Target shared model (e.g., Tag) |
| `morph` | `string` | ✅ | Morph name (determines junction table name + column prefix) |

### 5.5 `morph_by_many` (shared/inverse side — reads junction table)

```json
{
  "field_name": {
    "type": "morph_by_many",
    "model": "post",
    "morph": "taggable"
  }
}
```

| Property | Type | Required | Description |
|----------|------|:--------:|-------------|
| `type` | `"morph_by_many"` | ✅ | Many-to-many polymorphic (inverse/shared side) |
| `model` | `string` | ✅ | Target parent model (e.g., Post) |
| `morph` | `string` | ✅ | Morph name (must match the `morph` used in `morph_to_many` on the parent side) |

---

## 6. Database Schema

### 6.1 `morph_to` Columns

For field name `commentable`:

| Column | PostgreSQL | MySQL | SQLite | MongoDB |
|--------|-----------|-------|--------|---------|
| `commentable_type` | VARCHAR(255) | VARCHAR(255) | TEXT | String |
| `commentable_id` | VARCHAR(36) | VARCHAR(36) | TEXT | String |

**Why VARCHAR(36) for ID, not UUID?** Because the parent model's PK could be UUID, integer, or string (naming_series). VARCHAR(36) accommodates all PK types.

**Composite index**: `CREATE INDEX idx_{table}_commentable ON {table} (commentable_type, commentable_id)`

### 6.2 `morph_one` / `morph_many` Columns

No columns created — these are virtual. The columns exist on the child model's table (via `morph_to`).

### 6.3 `morph_to_many` Junction Table

For morph name `taggable`, between `post` and `tag`:

```sql
CREATE TABLE taggables (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tag_id UUID NOT NULL,
  taggable_id VARCHAR(36) NOT NULL,
  taggable_type VARCHAR(255) NOT NULL
);

CREATE INDEX idx_taggables_tag_id ON taggables (tag_id);
CREATE INDEX idx_taggables_morph ON taggables (taggable_type, taggable_id);
```

**Junction table naming**: `{morph_name}s` (pluralized morph name). `taggable` → `taggables`. This follows Laravel convention.

**Note**: The junction table is created once, shared by all models that use the same morph name. Post's tags and Video's tags both use the `taggables` table.

### 6.4 Column Naming Convention

```
morph_to field name: "commentable"
  → commentable_type (VARCHAR)
  → commentable_id (VARCHAR(36))

morph_to field name: "imageable"
  → imageable_type (VARCHAR)
  → imageable_id (VARCHAR(36))
```

The `_type` and `_id` suffixes are **always auto-appended**. Developer writes `"commentable"`, engine creates two columns.

---

## 7. Parser Changes

### 7.1 New FieldType Constants

```go
const (
    FieldMorphTo     FieldType = "morph_to"
    FieldMorphOne    FieldType = "morph_one"
    FieldMorphMany   FieldType = "morph_many"
    FieldMorphToMany FieldType = "morph_to_many"
    FieldMorphByMany FieldType = "morph_by_many"
)
```

### 7.2 New FieldDefinition Fields

```go
type FieldDefinition struct {
    // ... existing fields ...

    // Morph fields
    Morph   string   `json:"morph,omitempty"`    // morph_one/morph_many/morph_to_many/morph_by_many: morph name
    Models  []string `json:"models,omitempty"`   // morph_to: allowed parent models (optional, for validation + UI)
}
```

Note: `Inverse` field removed — replaced by explicit `morph_by_many` type.

### 7.3 Validation Rules

```go
// morph_to: no model required (that's the point)
if field.Type == FieldMorphTo {
    // models is optional — if set, used for validation + UI + GraphQL union generation
    // morph is NOT used on morph_to (it IS the morph)
}

// morph_one, morph_many: model + morph required
if field.Type == FieldMorphOne || field.Type == FieldMorphMany {
    if field.Model == "" {
        return error("morph_one/morph_many field %q must specify model")
    }
    if field.Morph == "" {
        return error("morph_one/morph_many field %q must specify morph (name of morph_to field on child model)")
    }
}

// morph_to_many: model + morph required
if field.Type == FieldMorphToMany {
    if field.Model == "" {
        return error("morph_to_many field %q must specify model")
    }
    if field.Morph == "" {
        return error("morph_to_many field %q must specify morph")
    }
}

// morph_by_many: model + morph required
if field.Type == FieldMorphByMany {
    if field.Model == "" {
        return error("morph_by_many field %q must specify model")
    }
    if field.Morph == "" {
        return error("morph_by_many field %q must specify morph")
    }
}
```

---

## 8. Migration: Column & Index Generation

### 8.1 `morph_to` — Create Two Columns + Composite Index

```go
func buildMorphToColumns(fieldName string, dialect DBDialect) string {
    var typeCol, idCol string
    switch dialect {
    case DialectPostgres:
        typeCol = "VARCHAR(255)"
        idCol = "VARCHAR(36)"
    case DialectMySQL:
        typeCol = "VARCHAR(255)"
        idCol = "VARCHAR(36)"
    default: // SQLite
        typeCol = "TEXT"
        idCol = "TEXT"
    }
    return fmt.Sprintf(
        "%s_type %s,\n  %s_id %s",
        fieldName, typeCol, fieldName, idCol,
    )
}
```

**Index**: Always create composite index on `(type, id)` for morph_to fields:

```sql
CREATE INDEX idx_{table}_{field}_morph ON {table} ({field}_type, {field}_id);
```

### 8.2 `morph_one` / `morph_many` — No Columns

These are virtual — no migration action needed. The columns are on the child table.

### 8.3 `morph_to_many` — Create Junction Table

```go
func createMorphJunctionTable(db *gorm.DB, morphName string, relatedModel string, dialect DBDialect, resolver TableNameResolver) error {
    tableName := inflection.Plural(morphName) // "taggable" → "taggables"

    if db.Migrator().HasTable(tableName) {
        return nil
    }

    relatedTable := resolver.TableName(relatedModel)
    relatedCol := relatedModel + "_id"

    var idType, fkType string
    switch dialect {
    case DialectPostgres:
        idType = "UUID PRIMARY KEY DEFAULT gen_random_uuid()"
        fkType = "UUID NOT NULL"
    case DialectMySQL:
        idType = "CHAR(36) PRIMARY KEY"
        fkType = "CHAR(36) NOT NULL"
    default:
        idType = "TEXT PRIMARY KEY"
        fkType = "TEXT NOT NULL"
    }

    sql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
  id %s,
  %s %s,
  %s_id VARCHAR(36) NOT NULL,
  %s_type VARCHAR(255) NOT NULL
)`, tableName, idType, relatedCol, fkType, morphName, morphName)

    if err := db.Exec(sql).Error; err != nil {
        return fmt.Errorf("failed to create morph junction table %s: %w", tableName, err)
    }

    // Indexes
    db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s (%s)", tableName, relatedCol, tableName, relatedCol))
    db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_morph ON %s (%s_type, %s_id)", tableName, tableName, morphName, morphName))

    return nil
}
```

### 8.4 Who Creates the Junction Table?

The **non-inverse side** (the parent model, e.g., Post) triggers junction table creation. The inverse side (Tag) does not create anything — it references the same table.

If only the inverse side is loaded first (Tag before Post), the junction table is created lazily when the first non-inverse model is registered.

---

## 9. Repository: Loading Morph Relations

### 9.1 `morph_to` — Load Parent (Dynamic Model)

```go
func (r *GenericRepository) loadMorphToRelation(ctx context.Context, w WithClause, results []map[string]any) {
    // 1. Group records by morph type
    //    { "post": [42, 55], "video": [7, 12] }
    typeGroups := make(map[string][]string)
    for _, rec := range results {
        morphType, _ := rec[w.Relation+"_type"].(string)
        morphID := fmt.Sprintf("%v", rec[w.Relation+"_id"])
        if morphType != "" && morphID != "" {
            typeGroups[morphType] = append(typeGroups[morphType], morphID)
        }
    }

    // 2. For each type, batch-load related records
    relatedByType := make(map[string]map[string]map[string]any)
    for modelName, ids := range typeGroups {
        table := r.resolveTableName(modelName)
        var related []map[string]any
        r.db.WithContext(ctx).Table(table).Where("id IN ?", unique(ids)).Find(&related)

        relatedMap := make(map[string]map[string]any)
        for _, rel := range related {
            relatedMap[fmt.Sprintf("%v", rel["id"])] = rel
        }
        relatedByType[modelName] = relatedMap
    }

    // 3. Attach to results
    for _, rec := range results {
        morphType, _ := rec[w.Relation+"_type"].(string)
        morphID := fmt.Sprintf("%v", rec[w.Relation+"_id"])
        if relMap, ok := relatedByType[morphType]; ok {
            if rel, ok := relMap[morphID]; ok {
                rec["_"+w.Relation] = rel
                rec["_"+w.Relation+"_type"] = morphType
            }
        }
    }
}
```

**Key insight**: Unlike `many2one` which queries ONE table, `morph_to` queries **N tables** (one per unique type in the result set). This is inherently more expensive but unavoidable.

### 9.2 `morph_one` / `morph_many` — Load Children (Filtered by Type)

```go
func (r *GenericRepository) loadMorphManyRelation(ctx context.Context, w WithClause, fieldDef parser.FieldDefinition, results []map[string]any) {
    // morph = "commentable" → query comments WHERE commentable_type = 'post' AND commentable_id IN (...)
    morphName := fieldDef.Morph
    parentType := r.morphType() // resolved via morph map or model name

    parentIDs := collectIDs(results, r.pkCol)
    if len(parentIDs) == 0 {
        return
    }

    relatedTable := r.resolveTableName(fieldDef.Model)
    var related []map[string]any
    r.db.WithContext(ctx).Table(relatedTable).
        Where(morphName+"_type = ? AND "+morphName+"_id IN ?", parentType, parentIDs).
        Find(&related)

    // Group by parent ID
    childMap := groupBy(related, morphName+"_id")

    // Attach
    for _, rec := range results {
        pid := fmt.Sprintf("%v", rec[r.pkCol])
        if fieldDef.Type == parser.FieldMorphOne {
            // morph_one: single record or nil
            if children, ok := childMap[pid]; ok && len(children) > 0 {
                rec["_"+w.Relation] = children[0]
            }
        } else {
            // morph_many: array
            rec["_"+w.Relation] = childMap[pid]
        }
    }
}
```

### 9.3 `morph_to_many` — Load via Junction Table

**`morph_to_many` (parent side — Post → Tags):**

```go
func (r *GenericRepository) loadMorphToManyRelation(ctx context.Context, w WithClause, fieldDef parser.FieldDefinition, results []map[string]any) {
    morphName := fieldDef.Morph
    junctionTable := inflection.Plural(morphName) // "taggable" → "taggables"
    parentType := r.morphType()
    relatedCol := fieldDef.Model + "_id"

    parentIDs := collectIDs(results, r.pkCol)
    if len(parentIDs) == 0 {
        return
    }

    // Query junction: WHERE taggable_type = 'post' AND taggable_id IN (...)
    var junctionRecords []map[string]any
    r.db.WithContext(ctx).Table(junctionTable).
        Where(morphName+"_type = ? AND "+morphName+"_id IN ?", parentType, parentIDs).
        Find(&junctionRecords)

    // Collect related IDs, batch-load from target table
    relatedTable := r.resolveTableName(fieldDef.Model)
    // ... (similar to many2many loading logic)
}
```

**`morph_by_many` (inverse side — Tag → Posts):**

```go
func (r *GenericRepository) loadMorphByManyRelation(ctx context.Context, w WithClause, fieldDef parser.FieldDefinition, results []map[string]any) {
    morphName := fieldDef.Morph
    junctionTable := inflection.Plural(morphName)
    relatedCol := r.modelName + "_id"

    parentIDs := collectIDs(results, r.pkCol)
    if len(parentIDs) == 0 {
        return
    }

    // Query junction: WHERE tag_id IN (...) AND taggable_type = 'post'
    targetType := r.morphMap.MorphType(fieldDef.Model)
    var junctionRecords []map[string]any
    r.db.WithContext(ctx).Table(junctionTable).
        Where(relatedCol+" IN ? AND "+morphName+"_type = ?", parentIDs, targetType).
        Find(&junctionRecords)

    // Collect morph IDs, batch-load from target model table
    relatedTable := r.resolveTableName(fieldDef.Model)
    // ... (similar to morph_to grouping logic but single type)
}
```

---

## 10. Bridge API: Morph Operations

### 10.1 Reading Morph Relations

Morph relations are loaded via the existing `with` parameter:

```javascript
// In script (any runtime)
const post = await bitcode.db.findOne("post", postId, {
  with: ["comments", "tags"]  // morph_many and morph_to_many loaded automatically
});

// post._comments = [{id: 1, body: "Great!", commentable_type: "post", commentable_id: "..."}]
// post._tags = [{id: 1, name: "javascript"}, {id: 2, name: "tutorial"}]
```

```javascript
// Load morph_to (parent from child)
const comment = await bitcode.db.findOne("comment", commentId, {
  with: ["commentable"]  // morph_to loaded — resolves to post or video
});

// comment._commentable = {id: 42, title: "My Post", ...}
// comment._commentable_type = "post"
```

### 10.2 Creating with Morph Relations

```javascript
// Create a comment on a post (morph_to)
await bitcode.db.create("comment", {
  body: "Great post!",
  commentable_type: "post",
  commentable_id: postId
});

// Attach tags to a post (morph_to_many)
await bitcode.db.morphAttach("post", postId, "tags", [tagId1, tagId2]);

// Detach tags
await bitcode.db.morphDetach("post", postId, "tags", [tagId1]);

// Sync tags (replace all)
await bitcode.db.morphSync("post", postId, "tags", [tagId1, tagId2, tagId3]);
```

### 10.3 New Bridge Methods

```
bitcode.db.morphAttach(model, id, relation, relatedIds)   → attach related records
bitcode.db.morphDetach(model, id, relation, relatedIds)    → detach related records
bitcode.db.morphSync(model, id, relation, relatedIds)      → sync (replace all)
```

These are only for `morph_to_many`. For `morph_to`, just set the `_type` and `_id` fields directly. For `morph_one`/`morph_many`, create/update the child record with the morph fields.

---

## 11. View Layer: Morph Fields in UI

### 11.1 `morph_to` — Two-Step Selector

In a form, `morph_to` renders as a **two-step selector**:

1. **Model selector** (dropdown): Choose the parent model type
2. **Record selector** (search dropdown): Choose the specific record

```
┌─────────────────────────────────┐
│ Commentable Type    [Post    ▼] │  ← Step 1: select model
│ Commentable Record  [Search...] │  ← Step 2: search records of selected model
└─────────────────────────────────┘
```

If `models` is specified in field definition, the model selector only shows those models. Otherwise, it shows all models.

### 11.2 `morph_one` — Embedded Form or Link

Similar to `one2many` but with a single record. Can render as:
- Inline embedded form (for simple child models like Image)
- Link to child record (for complex child models)

### 11.3 `morph_many` — Child Table

Same as `one2many` rendering — a table of child records with add/remove actions.

### 11.4 `morph_to_many` — Tag Input

Same as `many2many` rendering — tag input or multi-select.

### 11.5 Widget Mapping

```go
case parser.FieldMorphTo:
    return "bc-field-morph"      // new component: model selector + record selector
case parser.FieldMorphOne:
    return "bc-field-link"       // reuse existing link component
case parser.FieldMorphMany:
    return "bc-field-child-table" // reuse existing child table
case parser.FieldMorphToMany:
    return "bc-field-tags"       // reuse existing tags component
```

Only `morph_to` needs a new UI component. The others reuse existing components.

---

## 12. GraphQL: Morph Types

### 12.1 `morph_to` — Union Type

```graphql
union Commentable = Post | Video | Photo

type Comment {
  id: ID!
  body: String!
  commentable: Commentable
  commentable_type: String!
}
```

### 12.2 `morph_one` / `morph_many`

```graphql
type Post {
  id: ID!
  title: String!
  comments: [Comment!]!    # morph_many
}

type User {
  id: ID!
  name: String!
  avatar: Image            # morph_one
}
```

### 12.3 `morph_to_many` / `morph_by_many`

```graphql
type Post {
  id: ID!
  title: String!
  tags: [Tag!]!            # morph_to_many
}

type Tag {
  id: ID!
  name: String!
  posts: [Post!]!          # morph_by_many
  videos: [Video!]!        # morph_by_many
}
```

### 12.4 Union Type Generation

For `morph_to` fields, GraphQL type depends on whether `models` is specified:

**With `models` defined** — generate typed union:

```graphql
# morph_to with models: ["post", "video", "photo"]
union Commentable = Post | Video | Photo

type Comment {
  id: ID!
  body: String!
  commentable: Commentable
  commentable_type: String!
}
```

**Without `models` (unbounded)** — fallback to generic MorphRef:

```graphql
# morph_to without models (any model allowed)
type MorphRef {
  type: String!
  id: ID!
  data: JSON           # resolved record as JSON (optional, included when eager-loaded)
}

type ActivityLog {
  id: ID!
  action: String!
  subject: MorphRef    # could be any model
}
```

**Recommendation**: Always specify `models` on `morph_to` fields to get typed GraphQL unions. Omit only for truly unbounded use cases (activity logs, audit trails).

---

## 13. API: REST Endpoints for Morphs

### 13.1 Reading

Morph relations are included via `?with=` query parameter (same as existing relations):

```
GET /api/v1/blog/posts/42?with=comments,tags

Response:
{
  "id": "42",
  "title": "My Post",
  "_comments": [
    {"id": "1", "body": "Great!", "commentable_type": "post", "commentable_id": "42"}
  ],
  "_tags": [
    {"id": "1", "name": "javascript"}
  ]
}
```

### 13.2 Writing `morph_to`

Set morph fields directly in the request body:

```
POST /api/v1/blog/comments
{
  "body": "Great post!",
  "commentable_type": "post",
  "commentable_id": "42"
}
```

### 13.3 `morph_to_many` Attach/Detach/Sync

```
POST   /api/v1/blog/posts/42/tags/attach   { "ids": ["1", "2"] }
POST   /api/v1/blog/posts/42/tags/detach   { "ids": ["1"] }
POST   /api/v1/blog/posts/42/tags/sync     { "ids": ["1", "2", "3"] }
```

These endpoints follow the same pattern as existing `many2many` attach/detach.

---

## 14. Morph Map: Type Aliasing

### 14.1 Problem

By default, `commentable_type` stores the model name: `"post"`, `"video"`. But what if the model is renamed? Or what if you want shorter values?

### 14.2 Solution: Morph Map

Project-level configuration in `bitcode.toml`:

```toml
[morph_map]
post = "post"
video = "video"
# Or use short aliases:
# p = "post"
# v = "video"
```

Or in `module.json`:

```json
{
  "morph_map": {
    "post": "post",
    "video": "video"
  }
}
```

### 14.3 Default Behavior

If no morph map is configured, **model name is used as-is**. This is the simplest and most common case.

Morph map is a **nice-to-have** for advanced use cases (model renaming, shorter storage, cross-module aliasing).

### 14.4 Resolution

```go
func (r *Registry) MorphType(modelName string) string {
    // Check morph map first
    if alias, ok := r.morphMap[modelName]; ok {
        return alias
    }
    // Default: use model name
    return modelName
}

func (r *Registry) MorphModel(morphType string) string {
    // Reverse lookup
    for model, alias := range r.morphMap {
        if alias == morphType {
            return model
        }
    }
    // Default: morph type IS the model name
    return morphType
}
```

---

## 15. MongoDB Support

### 15.1 `morph_to` Columns

MongoDB stores morph fields as regular document fields:

```json
{
  "_id": ObjectId("..."),
  "body": "Great post!",
  "commentable_type": "post",
  "commentable_id": "42"
}
```

No special handling needed — MongoDB is schemaless.

### 15.2 `morph_to_many` Junction Collection

Junction "table" becomes a MongoDB collection:

```json
// taggables collection
{
  "_id": ObjectId("..."),
  "tag_id": "1",
  "taggable_id": "42",
  "taggable_type": "post"
}
```

### 15.3 Indexes

```javascript
// morph_to composite index
db.comments.createIndex({ commentable_type: 1, commentable_id: 1 })

// morph_to_many junction indexes
db.taggables.createIndex({ tag_id: 1 })
db.taggables.createIndex({ taggable_type: 1, taggable_id: 1 })
```

### 15.4 Loading

MongoDB morph loading follows the same logic as SQL — group by type, batch-load per type. The `MongoRepository` needs the same `loadMorphToRelation`, `loadMorphManyRelation`, etc. methods.

---

## 16. Edge Cases & Constraints

### 16.1 Cascade Delete

When a parent record is deleted, what happens to morph children?

**Current design**: No automatic cascade. Reasons:
- Morph relations are not FK-constrained (no `ON DELETE CASCADE` possible — the DB doesn't know about the polymorphic relationship)
- Cascade must be handled at application level via events

**Recommendation**: Use model events:

```json
{
  "name": "post",
  "events": {
    "before_delete": {
      "process": "cleanup_morph_children"
    }
  }
}
```

Or engine can provide a built-in `on_delete` option for morph fields in a future phase.

### 16.2 Orphaned Morph Records

If a parent is deleted without cleaning up morph children, orphaned records remain. Engine should provide:

1. **Detection**: `bitcode.db.morphOrphans("comment", "commentable")` — find comments whose parent no longer exists
2. **Cleanup**: Manual via script or scheduled job

This is a known trade-off of polymorphic relations in ALL frameworks (Laravel included).

### 16.3 Cross-Module Morph

```json
// Module A: blog
{ "name": "post", "fields": { "comments": { "type": "morph_many", "model": "comment", "morph": "commentable" } } }

// Module B: media
{ "name": "video", "fields": { "comments": { "type": "morph_many", "model": "comment", "morph": "commentable" } } }

// Module C: social
{ "name": "comment", "fields": { "commentable": { "type": "morph_to" } } }
```

This works because morph_to doesn't specify a fixed model. The `comment` model in module C doesn't need to know about `post` or `video`. It just stores `commentable_type` and `commentable_id`.

**Constraint**: The `comment` model must be loaded before or alongside the parent models. If `comment` is in a module that depends on `blog`, this is automatic.

### 16.4 Morph + Soft Deletes

If the parent model has soft deletes, morph loading should respect it:

```sql
-- When loading morph_to parent:
SELECT * FROM posts WHERE id IN (...) AND deleted_at IS NULL
```

This is already handled by the existing soft delete scope in the repository.

### 16.5 Morph + Record Rules

Record rules on the child model apply normally. Record rules on the parent model are NOT automatically applied when loading via morph_to (because the parent model is determined at runtime).

This is a documented limitation. Full morph-aware record rules may be added in a future phase.

### 16.6 Morph + Tenant Isolation

In multi-tenant mode, morph queries must include tenant filtering. Since morph loading goes through the repository (which already applies tenant scope), this is automatically handled.

---

## 17. Relationship with `dynamic_link`

### 17.1 `dynamic_link` vs `morph_to`

| Aspect | `dynamic_link` | `morph_to` |
|--------|---------------|------------|
| Target model | Fixed at definition time | Dynamic per record |
| Columns | 1 (FK) | 2 (`_type` + `_id`) |
| Use case | "This field links to model X" | "This field links to any model" |
| UI | Search dropdown for one model | Model selector + record selector |

### 17.2 Should `dynamic_link` Be Deprecated?

**No.** They serve different purposes:

- `dynamic_link` = "I know the model, I just want a different UI widget" (like a link instead of dropdown)
- `morph_to` = "I genuinely don't know the model until runtime"

However, `dynamic_link` currently requires `model` — which makes it identical to `many2one` with a different widget. Consider:

1. Keep `dynamic_link` as-is (UI variant of many2one)
2. If developer wants true dynamic linking → use `morph_to`

No changes to `dynamic_link` in this phase.

---

## 18. Implementation Tasks

### 18.1 Parser Layer

- [ ] Add FieldType constants: `morph_to`, `morph_one`, `morph_many`, `morph_to_many`, `morph_by_many`
- [ ] Add FieldDefinition fields: `morph`, `models`
- [ ] Add validation: morph_one/morph_many require `model` + `morph`
- [ ] Add validation: morph_to_many requires `model` + `morph`
- [ ] Add validation: morph_by_many requires `model` + `morph`
- [ ] Add validation: morph_to does not require `model`
- [ ] Write parser tests for all 5 morph types

### 18.2 Migration Layer

- [ ] `morph_to`: generate `_type` VARCHAR + `_id` VARCHAR(36) columns
- [ ] `morph_to`: generate composite index on `(_type, _id)`
- [ ] `morph_to_many`: generate junction table with morph columns
- [ ] `morph_to_many`: generate indexes on junction table
- [ ] `morph_one`/`morph_many`: no columns (virtual)
- [ ] MongoDB: create composite indexes for morph fields
- [ ] Write migration tests for all morph types × 4 databases

### 18.3 Repository Layer (SQL)

- [ ] Implement `loadMorphToRelation` — group by type, batch-load per type
- [ ] Implement `loadMorphOneRelation` — filter by type + parent IDs, single result
- [ ] Implement `loadMorphManyRelation` — filter by type + parent IDs, array result
- [ ] Implement `loadMorphToManyRelation` — junction table query (parent side)
- [ ] Implement `loadMorphByManyRelation` — junction table query (inverse/shared side)
- [ ] Add `morphAttach`, `morphDetach`, `morphSync` methods
- [ ] Update `WithClause` handling to detect morph field types
- [ ] Write repository tests for all morph loading patterns

### 18.4 Repository Layer (MongoDB)

- [ ] Implement morph loading in `MongoRepository` (same logic, different query syntax)
- [ ] Implement morph attach/detach/sync for MongoDB
- [ ] Write MongoDB morph tests

### 18.5 Bridge API

- [ ] Add `bitcode.db.morphAttach()` to bridge interface
- [ ] Add `bitcode.db.morphDetach()` to bridge interface
- [ ] Add `bitcode.db.morphSync()` to bridge interface
- [ ] Morph relations loadable via existing `with` parameter
- [ ] Write bridge tests for morph operations

### 18.6 View Layer

- [ ] Create `bc-field-morph` component (model selector + record selector)
- [ ] Update `component_compiler.go` — widget mapping for morph types
- [ ] `morph_one` → reuse `bc-field-link`
- [ ] `morph_many` → reuse child table rendering
- [ ] `morph_to_many` → reuse `bc-field-tags`
- [ ] Write view rendering tests

### 18.7 GraphQL

- [ ] Generate union types for `morph_to` fields WITH `models` defined
- [ ] Generate generic `MorphRef` type for `morph_to` fields WITHOUT `models`
- [ ] Handle `morph_one`/`morph_many` as regular nested types
- [ ] Handle `morph_to_many`/`morph_by_many` as regular list types
- [ ] Write GraphQL schema generation tests

### 18.8 REST API

- [ ] Morph data included in `?with=` responses
- [ ] `morph_to` fields writable via `_type` + `_id` in request body
- [ ] Add attach/detach/sync endpoints for `morph_to_many`
- [ ] Write API integration tests

### 18.9 Morph Map

- [ ] Add morph_map config support in `bitcode.toml`
- [ ] Add morph_map support in `module.json`
- [ ] Implement `MorphType()` and `MorphModel()` resolution
- [ ] Write morph map tests

### 18.10 Documentation

- [ ] Update `engine/docs/features/models.md` with morph relation types
- [ ] Add morph examples for each type
- [ ] Document morph map configuration
- [ ] Document limitations (no cascade, no morph record rules)
