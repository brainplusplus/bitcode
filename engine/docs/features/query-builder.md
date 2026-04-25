# Query Builder & OQL

## Overview

BitCode provides a comprehensive query builder that translates to both SQL (via GORM) and MongoDB (via official driver) with full parity. It supports 3 query interfaces:

1. **Programmatic Go API** — Fluent builder pattern
2. **JSON DSL** — For JSON definitions (process steps, API, views)
3. **OQL (Object Query Language)** — String-based query language with 3 syntax styles

## Query Builder (Go API)

### Basic WHERE

```go
q := persistence.NewQuery().
    Where("status", "=", "active").
    WhereGt("age", 18).
    WhereLike("email", "%@gmail.com").
    WhereIn("city", []string{"Jakarta", "Bandung"}).
    WhereBetween("created_at", "2024-01-01", "2024-12-31").
    WhereNull("deleted_at")
```

### OR Conditions

```go
q := persistence.NewQuery().
    Where("status", "=", "active").
    OrWhere("status", "=", "pending")
```

### Grouped Conditions

```go
// WHERE active = true AND (city = 'Jakarta' OR city = 'Bandung')
q := persistence.NewQuery().
    Where("active", "=", true).
    WhereGroup(func(sub *persistence.Query) *persistence.Query {
        return sub.Where("city", "=", "Jakarta").OrWhere("city", "=", "Bandung")
    })
```

### NOT

```go
q := persistence.NewQuery().
    WhereNot(func(sub *persistence.Query) *persistence.Query {
        return sub.Where("status", "=", "deleted")
    })
```

### JOINs

```go
q := persistence.NewQuery().
    InnerJoin("companies", "contacts.company_id", "companies.id").
    LeftJoin("addresses", "contacts.id", "addresses.contact_id").
    RightJoin("departments", "contacts.dept_id", "departments.id").
    CrossJoin("settings").
    FullJoin("profiles", "contacts.id", "profiles.contact_id")
```

### Aggregates

```go
q := persistence.NewQuery().
    SetGroupBy("department").
    SelectCount("*", "total").
    SelectSum("salary", "total_salary").
    SelectAvg("age", "avg_age").
    SelectMin("salary", "min_salary").
    SelectMax("salary", "max_salary").
    SelectCountDistinct("city", "unique_cities").
    HavingCondition("COUNT", "*", ">", 5)
```

### Eager Loading (WITH / Preload)

```go
q := persistence.NewQuery().
    WithRelation("company").
    WithRelation("tags").
    WithRelationConditions("orders", []persistence.Condition{
        {Field: "status", Operator: "=", Value: "active"},
    })
```

### UNION

```go
q1 := persistence.NewQuery().Where("type", "=", "A")
q2 := persistence.NewQuery().Where("type", "=", "B")
q := q1.Union(q2)       // UNION
q := q1.UnionAll(q2)    // UNION ALL
```

### Subqueries

```go
sub := persistence.NewQuery().SetSelect("id").Where("active", "=", true)
q := persistence.NewQuery().WhereInSubQuery("company_id", sub)
q := persistence.NewQuery().WhereExistsQuery(sub)
```

### Locking

```go
q := persistence.NewQuery().LockForUpdate()
q := persistence.NewQuery().LockForShare()
```

### Soft Delete Scopes

```go
q := persistence.NewQuery().WithTrashed()    // include deleted
q := persistence.NewQuery().OnlyTrashed()    // only deleted
```

### Scopes (Reusable)

```go
activeScope := func(q *persistence.Query) *persistence.Query {
    return q.Where("active", "=", true)
}
q := persistence.NewQuery().Scope(activeScope)
```

### Raw Expressions

```go
q := persistence.NewQuery().
    WhereRawExpr("age > ? AND age < ?", 18, 65).
    SelectRawExpr("COUNT(*) as total").
    HavingRawExpr("COUNT(*) > ?", 5)
```

## Repository Methods

| Method | Description |
|--------|-------------|
| `FindAll(ctx, query, page, pageSize)` | Paginated query |
| `FindAllActive(ctx, query, page, pageSize)` | Active-only query |
| `FindAllWithTrashed(ctx, query, page, pageSize)` | Include soft-deleted |
| `FindAllOnlyTrashed(ctx, query, page, pageSize)` | Only soft-deleted |
| `Count(ctx, query)` | Count matching |
| `Sum(ctx, field, query)` | Sum field |
| `Avg(ctx, field, query)` | Average field |
| `Min(ctx, field, query)` | Minimum field |
| `Max(ctx, field, query)` | Maximum field |
| `Pluck(ctx, field, query)` | Extract single column |
| `Exists(ctx, query)` | Check existence |
| `Aggregate(ctx, query)` | Grouped aggregate results |
| `Chunk(ctx, query, size, fn)` | Batch processing |
| `Increment(ctx, id, field, value)` | Increment field |
| `Decrement(ctx, id, field, value)` | Decrement field |
| `Transaction(ctx, fn)` | Transactional execution |
| `RawQuery(ctx, sql, values...)` | Raw SQL query |
| `RawExec(ctx, sql, values...)` | Raw SQL execute |

## Model Process Built-in Functions

All accessible via `models.{name}.{operation}`:

| Operation | Args |
|-----------|------|
| `Get` / `Find` | `id` |
| `GetAll` / `FindAll` | `wheres`, `order`, `page`, `page_size`, `oql` |
| `Paginate` | same as FindAll |
| `Create` | `data` |
| `Update` | `id`, `data` |
| `Delete` | `id` |
| `Upsert` | `data`, `unique` |
| `Count` / `CountActive` | `wheres`, `oql` |
| `Sum` / `SumActive` | `field`, `wheres`, `oql` |
| `Avg` | `field`, `wheres`, `oql` |
| `Min` | `field`, `wheres`, `oql` |
| `Max` | `field`, `wheres`, `oql` |
| `Pluck` | `field`, `wheres`, `oql` |
| `Exists` | `wheres`, `oql` |
| `Aggregate` | `wheres`, `group_by`, `aggregates`, `oql` |
| `WithTrashed` | `wheres`, `page`, `page_size`, `oql` |
| `OnlyTrashed` | `wheres`, `page`, `page_size`, `oql` |
| `Increment` | `id`, `field`, `value` |
| `Decrement` | `id`, `field`, `value` |

## Dynamic Finders

Convention-based query methods parsed from operation name. Available via `models.{name}.{operation}`:

### Patterns

| Pattern | Example | SQL |
|---------|---------|-----|
| `FindBy{Field}` | `FindByEmail` | `WHERE email = ? LIMIT 1` |
| `FindAllBy{Field}` | `FindAllByStatus` | `WHERE status = ?` |
| `FindBy{Field}And{Field}` | `FindByStatusAndCity` | `WHERE status = ? AND city = ?` |
| `FindBy{Field}Or{Field}` | `FindByEmailOrPhone` | `WHERE email = ? OR phone = ?` |
| `CountBy{Field}` | `CountByStatus` | `COUNT WHERE status = ?` |
| `ExistsBy{Field}` | `ExistsByEmail` | `EXISTS WHERE email = ?` |
| `DeleteBy{Field}` | `DeleteByStatus` | `DELETE WHERE status = ?` |
| `SumBy{AggField}{Field}` | `SumByAmountStatus` | `SUM(amount) WHERE status = ?` |
| `AvgBy{AggField}{Field}` | `AvgByAgeStatus` | `AVG(age) WHERE status = ?` |
| `MinBy{AggField}{Field}` | `MinByPriceCategory` | `MIN(price) WHERE category = ?` |
| `MaxBy{AggField}{Field}` | `MaxBySalaryDepartment` | `MAX(salary) WHERE department = ?` |
| `PluckBy{AggField}{Field}` | `PluckByEmailStatus` | `SELECT email WHERE status = ?` |

### Operator Suffixes

| Suffix | Operator | Example |
|--------|----------|---------|
| (none) | `=` | `FindByStatus` |
| `Gt` | `>` | `FindAllByAgeGt` |
| `Gte` | `>=` | `FindAllByAgeGte` |
| `Lt` | `<` | `FindAllByPriceLt` |
| `Lte` | `<=` | `FindAllByPriceLte` |
| `Not` | `!=` | `FindAllByStatusNot` |
| `Like` | `LIKE` | `FindAllByNameLike` |
| `In` | `IN` | `FindAllByCityIn` |
| `NotIn` | `NOT IN` | `FindAllByStatusNotIn` |
| `Between` | `BETWEEN` | `FindAllByAgeBetween` |
| `IsNull` | `IS NULL` | `FindAllByDeletedAtIsNull` |
| `IsNotNull` | `IS NOT NULL` | `FindAllByEmailIsNotNull` |

### OrderBy

Append `OrderBy{Field}Asc` or `OrderBy{Field}Desc`:

```
FindAllByStatusOrderByNameAsc
FindAllByActiveOrderByCreatedAtDesc
FindAllByStatusOrderByNameAscAndCreatedAtDesc
```

### Usage in JSON

```json
{
  "process": "models.contacts.FindAllByStatusOrderByNameAsc",
  "args": { "status": "active" }
}
```

```json
{
  "process": "models.orders.SumByAmountStatus",
  "args": { "status": "confirmed" }
}
```

## OQL (Object Query Language)

Three syntax styles, auto-detected by `ParseOQL()`:

### Style A: SQL-like (JPQL/HQL)

```
SELECT name, email FROM contacts
  LEFT JOIN companies ON contacts.company_id = companies.id
  WHERE status = 'active' AND (city = 'Jakarta' OR city = 'Bandung')
  ORDER BY name ASC
  LIMIT 10
  WITH company, tags
```

### Style B: Simplified DSL

```
contacts[status='active', city IN ('Jakarta','Bandung')] ORDER BY name WITH company LIMIT 10
```

### Style C: Dot-notation

```
contacts.where(status.eq('active')).where(city.in('Jakarta','Bandung')).orderBy('name').with('company').limit(10)
```

### Using OQL in JSON Definitions

Process step:
```json
{
  "type": "query",
  "model": "contacts",
  "oql": "contacts[status='active'] ORDER BY name WITH company",
  "into": "active_contacts"
}
```

Model process call:
```json
{
  "process": "models.contacts.FindAll",
  "args": {
    "oql": "SELECT * FROM contacts WHERE status = 'active' WITH company",
    "page": 1,
    "page_size": 20
  }
}
```

API query parameter:
```
GET /api/contacts?oql=contacts[status='active']&page=1&page_size=20
```

## JSON DSL

Full query as JSON:

```json
{
  "wheres": [
    {"field": "status", "op": "=", "value": "active"}
  ],
  "where_groups": [
    {
      "connector": "OR",
      "conditions": [
        {"field": "city", "op": "=", "value": "Jakarta"},
        {"field": "city", "op": "=", "value": "Bandung"}
      ]
    }
  ],
  "joins": [
    {"type": "LEFT", "table": "companies", "local_key": "contacts.company_id", "foreign_key": "companies.id"}
  ],
  "select": ["name", "email"],
  "order": [{"field": "name", "direction": "asc"}],
  "group_by": ["department"],
  "having": [{"aggregate": "COUNT", "field": "*", "op": ">", "value": 5}],
  "distinct": true,
  "with": ["company", {"relation": "orders", "conditions": [{"field": "status", "op": "=", "value": "active"}]}],
  "limit": 10,
  "offset": 20,
  "lock": "for_update",
  "soft_delete_scope": "with_trashed"
}
```

## Supported Operators

| Operator | SQL | MongoDB |
|----------|-----|---------|
| `=` | `field = ?` | `{field: value}` |
| `!=` | `field != ?` | `{field: {$ne: value}}` |
| `>` | `field > ?` | `{field: {$gt: value}}` |
| `<` | `field < ?` | `{field: {$lt: value}}` |
| `>=` | `field >= ?` | `{field: {$gte: value}}` |
| `<=` | `field <= ?` | `{field: {$lte: value}}` |
| `like` | `field LIKE ?` | `{field: {$regex: pattern}}` |
| `not_like` | `field NOT LIKE ?` | `{field: {$not: {$regex}}}` |
| `in` | `field IN (?)` | `{field: {$in: [...]}}` |
| `not_in` | `field NOT IN (?)` | `{field: {$nin: [...]}}` |
| `between` | `field BETWEEN ? AND ?` | `{field: {$gte: a, $lte: b}}` |
| `not_between` | `field NOT BETWEEN ? AND ?` | `{$or: [{$lt: a}, {$gt: b}]}` |
| `is_null` | `field IS NULL` | `{field: {$eq: null}}` |
| `is_not_null` | `field IS NOT NULL` | `{field: {$ne: null}}` |
| `column:op` | `field1 op field2` | N/A (SQL only) |

## SQL/MongoDB Parity

Both drivers support:
- All operators above
- OR/AND/NOT condition groups
- Ordering and pagination
- Count, Sum, Avg, Min, Max
- Pluck, Exists
- Grouped aggregates
- Soft delete scopes
- Transactions
- Increment/Decrement
- Chunk processing

SQL-only features:
- JOINs (MongoDB uses `$lookup` in Aggregate pipeline)
- HAVING
- DISTINCT
- Subqueries
- UNION
- Raw SQL
- Locking (FOR UPDATE/FOR SHARE)
- Column comparison (`whereColumn`)
