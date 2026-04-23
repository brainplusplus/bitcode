# File Upload & Storage System — Design Document

**Date**: 2026-04-24
**Status**: Approved
**Scope**: Engine (Go) + Stencil Component

---

## Problem

Current file upload is minimal: single file, local-only, no database tracking, no S3, no path formatting, no versioning. Insufficient for ERP use cases where files are business-critical documents (invoices, contracts, receipts).

## Goals

1. Single and multiple file upload via one component (`bc-field-file` with `multiple` prop)
2. Configurable storage backend: local filesystem or S3 (selected via config)
3. Flexible path/name formatting with template variables (session, model, time, record data)
4. `attachments` database table tracking all uploaded files
5. File versioning with configurable max versions
6. Thumbnail generation for images with on-demand resize API
7. Security: per-file public/private flag, auth middleware for local, signed URLs for S3
8. Duplicate detection via SHA256 hash
9. Orphan file cleanup via cron agent
10. Per-model file config override in `model.json`
11. `file` field type auto-links to attachment table

## Non-Goals (YAGNI)

- CDN integration
- Image cropping/editing
- Watermarking
- Video transcoding

---

## Configuration

### Storage Config (Viper)

Priority: `.toml` > `.yaml` > `.env` > OS environment variables.

```toml
# bitcode.toml
[storage]
driver = "local"                    # "local" | "s3"
max_size = 10485760                 # 10MB default (bytes)
allowed_extensions = [".jpg", ".jpeg", ".png", ".gif", ".webp", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".csv", ".txt", ".zip"]
path_format = "{model}/{year}/{month}"
name_format = "{uuid}_{original}{ext}"

[storage.local]
path = "uploads"
base_url = "/uploads"

[storage.s3]
bucket = ""
region = ""
endpoint = ""                       # custom endpoint for MinIO, DigitalOcean Spaces, etc.
access_key = ""                     # fallback: AWS_ACCESS_KEY_ID env
secret_key = ""                     # fallback: AWS_SECRET_ACCESS_KEY env
use_path_style = false              # true for MinIO
signed_url_expiry = 3600            # seconds (1 hour default)

[storage.thumbnail]
enabled = true
width = 300
height = 300
quality = 85
```

### Per-Model Override (model.json)

```json
{
  "name": "invoice",
  "file_config": {
    "max_size": 5242880,
    "allowed_extensions": [".pdf", ".jpg", ".png"]
  }
}
```

### Settings Table Entries

| Key | Default | Description |
|-----|---------|-------------|
| `storage.max_versions` | `5` | Max file versions per attachment |
| `storage.thumbnail.width` | `300` | Default thumbnail width |
| `storage.thumbnail.height` | `300` | Default thumbnail height |
| `storage.thumbnail.quality` | `85` | JPEG quality (1-100) |

---

## Database Schema

### `attachments` Table

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| `id` | UUID | NO | gen | Primary key |
| `tenant_id` | UUID | YES | NULL | Multi-tenant isolation |
| `user_id` | UUID | YES | NULL | FK → users. Uploader |
| `model` | VARCHAR(255) | YES | NULL | Linked model name |
| `record_id` | UUID | YES | NULL | Linked record ID |
| `field_name` | VARCHAR(255) | YES | NULL | Linked field name |
| `name` | VARCHAR(500) | NO | — | Original filename (sanitized) |
| `path` | VARCHAR(1000) | NO | — | Storage path/key |
| `url` | VARCHAR(1000) | NO | — | Accessible URL |
| `storage` | VARCHAR(20) | NO | — | `local` or `s3` |
| `size` | BIGINT | NO | 0 | File size in bytes |
| `mime_type` | VARCHAR(255) | NO | — | MIME type |
| `ext` | VARCHAR(20) | NO | — | File extension |
| `hash` | VARCHAR(64) | NO | — | SHA256 checksum |
| `is_public` | BOOLEAN | NO | false | Public access flag |
| `version` | INTEGER | NO | 1 | Version number |
| `parent_id` | UUID | YES | NULL | FK → attachments. Original file (versioning) |
| `thumbnail_path` | VARCHAR(1000) | YES | NULL | Thumbnail storage path |
| `metadata` | JSON | YES | NULL | Extra data (width, height, duration, etc.) |
| `active` | BOOLEAN | NO | true | Soft delete flag |
| `created_at` | DATETIME | NO | now | Upload timestamp |
| `updated_at` | DATETIME | NO | now | Last update |

### Indexes

- `idx_attachments_model_record` on (`model`, `record_id`, `field_name`)
- `idx_attachments_hash` on (`hash`)
- `idx_attachments_tenant` on (`tenant_id`)
- `idx_attachments_parent` on (`parent_id`)
- `idx_attachments_user` on (`user_id`)

---

## Template Variables (Path/Name Formatting)

### Standard Variables

| Variable | Source | Example |
|----------|--------|---------|
| `{tenant_id}` | Session | `tenant-abc` |
| `{user_id}` | Session | `usr_123` |
| `{model}` | Upload context | `invoice` |
| `{year}` | Current time | `2026` |
| `{month}` | Current time | `04` |
| `{day}` | Current time | `24` |
| `{date}` | Current time | `2026-04-24` |
| `{timestamp}` | Current time | `20260424_153045` |
| `{uuid}` | Generated | `a1b2c3d4-e5f6-...` |
| `{original}` | Original filename (no ext, sanitized) | `invoice_april` |
| `{ext}` | Original extension | `.pdf` |

### Dynamic Variables

| Variable | Source | Example |
|----------|--------|---------|
| `{input.xxx}` | Form input field value | `{input.customer_name}` → `acme_corp` |
| `{data.xxx}` | Model record field value | `{data.invoice_number}` → `INV-2026-001` |

### Resolution

1. Parse template string for `{...}` tokens
2. Resolve standard variables from context (session, time)
3. Resolve `{input.xxx}` from form submission data
4. Resolve `{data.xxx}` from linked model record
5. Sanitize all values (remove path separators, special chars)
6. Unresolved variables → empty string with warning log

---

## Storage Interface

```go
type StorageDriver interface {
    Put(ctx context.Context, path string, reader io.Reader, opts PutOptions) error
    Get(ctx context.Context, path string) (io.ReadCloser, error)
    Delete(ctx context.Context, path string) error
    URL(ctx context.Context, path string, opts URLOptions) (string, error)
    Exists(ctx context.Context, path string) (bool, error)
}

type PutOptions struct {
    ContentType string
    Metadata    map[string]string
    IsPublic    bool
}

type URLOptions struct {
    Expiry   time.Duration  // for signed URLs (S3)
    IsPublic bool
}
```

### Scan Hook

```go
type ScanHook interface {
    BeforePut(ctx context.Context, filename string, reader io.Reader) (io.Reader, error)
}
```

Not implemented now, but the hook point exists in the upload flow. Future: ClamAV, external scanner.

---

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/upload` | Yes | Single file upload |
| `POST` | `/api/uploads` | Yes | Multiple file upload |
| `GET` | `/api/files/:id` | Yes* | File metadata |
| `GET` | `/api/files/:id/download` | Yes* | Download file |
| `GET` | `/api/files/:id/thumbnail` | Yes* | Get thumbnail |
| `GET` | `/api/files/:id/resize` | Yes* | On-demand resize (`?w=200&h=200`) |
| `GET` | `/api/files` | Yes | List files (filter: `model`, `record_id`, `field_name`) |
| `DELETE` | `/api/files/:id` | Yes | Soft delete |
| `GET` | `/api/files/:id/versions` | Yes | List versions |

*Yes* = auth required unless `is_public = true`.

### Upload Request

```
POST /api/upload
Content-Type: multipart/form-data

file: (binary)
model: invoice          (optional)
record_id: uuid         (optional)
field_name: attachments (optional)
is_public: false        (optional)
metadata: {}            (optional JSON)
```

### Upload Response

```json
{
  "id": "uuid",
  "name": "invoice_april.pdf",
  "url": "/api/files/uuid/download",
  "size": 1048576,
  "mime_type": "application/pdf",
  "ext": ".pdf",
  "hash": "sha256...",
  "version": 1,
  "thumbnail_url": null,
  "created_at": "2026-04-24T15:30:45Z"
}
```

### Multiple Upload Response

```json
{
  "files": [ ...array of single upload responses... ],
  "total": 3,
  "errors": []
}
```

---

## Duplicate Detection

On upload:
1. Compute SHA256 hash
2. Query `attachments` WHERE `hash = ? AND model = ? AND record_id = ? AND field_name = ? AND active = true`
3. If match found → return existing attachment (no re-upload)
4. If no match → proceed with upload

---

## File Versioning

On upload with existing `model` + `record_id` + `field_name` + `parent_id`:
1. Find latest version for that combination
2. New version = latest + 1
3. If version count > `storage.max_versions` setting → soft-delete oldest version
4. Old file remains accessible via `/api/files/:id/versions`

---

## Orphan Cleanup

Cron agent (configurable interval, default 24h):
1. Query attachments WHERE `record_id IS NULL AND created_at < NOW() - 24h AND active = true`
2. Delete file from storage
3. Hard-delete attachment record

---

## Thumbnail Generation

On image upload (MIME type `image/*`):
1. Read image
2. Resize to configured dimensions (default 300x300, maintain aspect ratio)
3. Save to `{path}_thumb{ext}`
4. Store `thumbnail_path` in attachment record

### On-Demand Resize

`GET /api/files/:id/resize?w=200&h=200`
1. Validate dimensions (max 2000x2000)
2. Generate resized image
3. Cache in storage as `{path}_w200_h200{ext}`
4. Return resized image

---

## File Layer Map

| Layer | Path | Responsibility |
|-------|------|----------------|
| Domain | `internal/domain/storage/attachment.go` | Attachment entity, StorageDriver interface, ScanHook interface |
| Infrastructure | `internal/infrastructure/storage/local.go` | Local filesystem driver |
| Infrastructure | `internal/infrastructure/storage/s3.go` | S3 driver |
| Infrastructure | `internal/infrastructure/storage/formatter.go` | Path/name template formatter |
| Infrastructure | `internal/infrastructure/storage/thumbnail.go` | Image thumbnail + on-demand resize |
| Infrastructure | `internal/infrastructure/storage/repository.go` | GORM repository for attachments |
| Presentation | `internal/presentation/api/file_handler.go` | All file API endpoints |
| Config | Viper | `[storage]` section |
| Component | `packages/components/src/components/bc-field-file/` | Stencil upload component |

---

## Security

- **Private files** (default): Auth middleware checks user permission on linked model
- **Public files**: `is_public = true` → no auth required
- **S3**: Signed URLs with configurable expiry (default 1 hour)
- **Local**: Served through handler (NOT static), auth checked per request
- **Validation**: File size (global + per-model), extension whitelist, MIME type verification
- **Sanitization**: Filename sanitized (no path traversal, special chars stripped)

---

## Stencil Component: `bc-field-file`

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `name` | string | — | Field name |
| `value` | string/string[] | — | Current file ID(s) |
| `multiple` | boolean | false | Allow multiple files |
| `accept` | string | — | Accepted MIME types |
| `max-size` | number | — | Max file size (bytes) |
| `disabled` | boolean | false | Disable upload |
| `api-base` | string | `/api` | API base URL |

### Features

- Drag & drop zone
- Click to browse
- Upload progress bar
- Image preview (thumbnail)
- File icon for non-images
- File list with remove button
- Error display (size, type validation)
