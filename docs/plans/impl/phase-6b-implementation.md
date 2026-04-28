# Phase 6B Implementation Plan: Polymorphic Relations (Morphs)

**Estimated effort**: 7-9 days
**Prerequisites**: Phase 6A (new field types, storage hints)
**Test command**: `go test ./internal/compiler/parser/... ./internal/infrastructure/persistence/... ./internal/domain/model/...`

---

## Implementation Order

```
Stream 1: Parser — 5 Morph Types (Day 1-2)
  ↓
Stream 2: Migration — Morph Columns & Junction Tables (Day 2-3)
  ↓
Stream 3: Repository — Morph Loading (Day 3-5)
  ↓
Stream 4: Repository — morphAttach/Detach/Sync (Day 5-6)
  ↓
Stream 5: Bridge, View, GraphQL, API (Day 6-8)
  ↓
Stream 6: Morph Map & Tests (Day 8-9)
```

---

## Stream 1: Parser

**File**: `internal/compiler/parser/model.go`

### 1.1 Add FieldType Constants

```go
FieldMorphTo     FieldType = "morph_to"
FieldMorphOne    FieldType = "morph_one"
FieldMorphMany   FieldType = "morph_many"
FieldMorphToMany FieldType = "morph_to_many"
FieldMorphByMany FieldType = "morph_by_many"
```

### 1.2 Add Fields to FieldDefinition

```go
Morph   string   `json:"morph,omitempty"`
Models  []string `json:"models,omitempty"`
```

### 1.3 Validation Rules

- `morph_to`: no `model` required, `models` optional
- `morph_one`, `morph_many`: `model` + `morph` required
- `morph_to_many`: `model` + `morph` required
- `morph_by_many`: `model` + `morph` required

---

## Stream 2: Migration

**File**: `internal/infrastructure/persistence/dynamic_model.go`

### 2.1 morph_to Columns

In `buildColumns()`, when field type is `morph_to`:
- Generate `{field}_type VARCHAR(255)` column
- Generate `{field}_id VARCHAR(36)` column
- Generate composite index on `({field}_type, {field}_id)`

### 2.2 morph_to_many Junction Table

New function `createMorphJunctionTable()`:
- Table name: `inflection.Plural(morph)` (e.g., "taggable" → "taggables")
- Columns: `id`, `{related_model}_id`, `{morph}_id`, `{morph}_type`
- Indexes on `{related_model}_id` and `({morph}_type, {morph}_id)`

### 2.3 morph_one, morph_many, morph_by_many

No columns — virtual fields. Return empty string from `fieldTypeToSQL`.

---

## Stream 3: Repository — Loading

**File**: `internal/infrastructure/persistence/repository.go`

### 3.1 Update loadWithRelations

Add cases for morph types in the field type switch:
```go
case parser.FieldMorphTo:
    r.loadMorphToRelation(ctx, w, results)
case parser.FieldMorphOne:
    r.loadMorphOneRelation(ctx, w, fieldDef, results)
case parser.FieldMorphMany:
    r.loadMorphManyRelation(ctx, w, fieldDef, results)
case parser.FieldMorphToMany:
    r.loadMorphToManyRelation(ctx, w, fieldDef, results)
case parser.FieldMorphByMany:
    r.loadMorphByManyRelation(ctx, w, fieldDef, results)
```

### 3.2 Implement Each Loader

- `loadMorphToRelation` — group by `_type`, batch-load per type from different tables
- `loadMorphOneRelation` — query child table WHERE `{morph}_type = parentType AND {morph}_id IN (...)`
- `loadMorphManyRelation` — same as morph_one but returns array
- `loadMorphToManyRelation` — query junction table, then batch-load related records
- `loadMorphByManyRelation` — query junction table by related model ID, filter by target type

---

## Stream 4: Attach/Detach/Sync

**File**: `internal/infrastructure/persistence/repository.go`

```go
func (r *GenericRepository) MorphAttach(ctx, morphName, relatedModel, parentID string, relatedIDs []string) error
func (r *GenericRepository) MorphDetach(ctx, morphName, relatedModel, parentID string, relatedIDs []string) error
func (r *GenericRepository) MorphSync(ctx, morphName, relatedModel, parentID string, relatedIDs []string) error
```

---

## Stream 5: Bridge, View, GraphQL, API

- **Bridge**: Add `bitcode.db.morphAttach/Detach/Sync` methods
- **View**: Create `bc-field-morph` component (model selector + record selector)
- **GraphQL**: Generate union types for morph_to with `models`, MorphRef for unbounded
- **API**: Add attach/detach/sync endpoints for morph_to_many

---

## Stream 6: Morph Map

- Parse `morph_map` from `bitcode.toml` and `module.json`
- `MorphType(modelName)` and `MorphModel(morphType)` resolution

## Definition of Done

- [ ] 5 morph types parse correctly
- [ ] morph_to creates two columns + composite index
- [ ] morph_to_many creates junction table
- [ ] All 5 morph loaders work in repository
- [ ] morphAttach/Detach/Sync work
- [ ] Bridge API exposes morph operations
- [ ] GraphQL generates union types (with models) or MorphRef (without)
- [ ] Morph map works for type aliasing
- [ ] MongoDB morph loading works
- [ ] All tests pass
