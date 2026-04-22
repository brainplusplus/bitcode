# Models

Models define your data structure. Each model becomes a database table.

## Minimal Example

```json
{
  "name": "customer",
  "fields": {
    "name":  { "type": "string", "required": true },
    "email": { "type": "email", "unique": true }
  }
}
```

This creates a `customers` table with `id`, `name`, `email`, `created_at`, `updated_at`, `created_by`, `updated_by`, `active` columns.

## Field Types

| Type | SQLite | PostgreSQL | MySQL | Description |
|------|--------|-----------|-------|-------------|
| `string` | TEXT | VARCHAR(n) | VARCHAR(n) | Short text. Use `max` for length. |
| `text` | TEXT | TEXT | TEXT | Long text |
| `integer` | INTEGER | INTEGER | INTEGER | Whole number. `min`/`max` for range. |
| `decimal` | REAL | DECIMAL(18,n) | DECIMAL(18,n) | Number with decimals. `precision` for decimal places. |
| `boolean` | INTEGER | BOOLEAN | BOOLEAN | True/false |
| `date` | TEXT | DATE | DATE | Date only |
| `datetime` | TEXT | TIMESTAMPTZ | DATETIME | Date + time |
| `selection` | TEXT | VARCHAR(50) | VARCHAR(50) | Enum. Requires `options` array. |
| `email` | TEXT | VARCHAR(255) | VARCHAR(255) | Email (validated) |
| `many2one` | TEXT | UUID | CHAR(36) | FK to another model. Requires `model`. |
| `one2many` | - | - | - | Reverse FK. Requires `model` + `inverse`. No column created. |
| `many2many` | - | - | - | Junction table. Requires `model`. |
| `json` | TEXT | JSONB | JSON | Arbitrary JSON data |
| `file` | TEXT | VARCHAR(500) | VARCHAR(500) | File path/URL |
| `computed` | - | - | - | Virtual field. Requires `computed` expression. No column created. |

## Field Options

| Option | Type | Description |
|--------|------|-------------|
| `required` | bool | NOT NULL constraint |
| `unique` | bool | UNIQUE constraint |
| `default` | any | Default value. Use `"now"` for current timestamp. |
| `max` | int | Max length (string) or max value (integer) |
| `min` | int | Min value (integer) |
| `precision` | int | Decimal places (decimal) |
| `max_size` | string | Max file size, e.g. `"5MB"` (file) |
| `label` | string | Display label for UI |
| `options` | string[] | Enum values (selection) |
| `model` | string | Related model name (many2one, one2many, many2many) |
| `inverse` | string | Inverse FK field name (one2many) |
| `computed` | string | Computation expression (computed) |
| `auto` | bool | Auto-set value (datetime) |

## Auto-generated Columns

Every model automatically gets:

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key (auto-generated) |
| `created_at` | datetime | Set on creation |
| `updated_at` | datetime | Set on every update |
| `created_by` | UUID FK | User who created the record |
| `updated_by` | UUID FK | User who last updated |
| `active` | boolean | Soft delete flag (default true) |

You never need to define these in your JSON.

## Relationships

### many2one (FK)

```json
"customer_id": { "type": "many2one", "model": "customer", "required": true }
```

Creates a UUID foreign key column.

### one2many (Reverse FK)

```json
"lines": { "type": "one2many", "model": "order_line", "inverse": "order_id" }
```

No column created. Resolved by querying `order_lines WHERE order_id = this.id`.

### many2many (Junction Table)

```json
"tags": { "type": "many2many", "model": "tag" }
```

Creates a junction table `model_tags` with both FKs.

## Record Rules (Row-Level Security)

```json
"record_rules": [
  { "groups": ["sales.user"],    "domain": [["created_by", "=", "{{user.id}}"]] },
  { "groups": ["sales.manager"], "domain": [] }
]
```

- `sales.user` group: can only see records they created
- `sales.manager` group: can see all records (empty domain = no filter)

Record rules are enforced automatically when `auth: true` on the API.

## Model Inheritance

```json
{
  "name": "vip_customer",
  "inherit": "customer",
  "fields": {
    "vip_level": { "type": "selection", "options": ["gold", "platinum"] },
    "discount_rate": { "type": "decimal", "default": 0.1 }
  }
}
```

Adds fields to the parent model's table.

## Indexes

```json
"indexes": [
  ["customer_id", "order_date"],
  ["status"]
]
```

Creates database indexes for query performance.
