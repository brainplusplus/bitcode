# File Upload & Storage

The storage system handles single and multiple file uploads, stores file metadata in `attachments`, supports local filesystem and S3 backends, generates thumbnails for images, tracks versions, detects duplicates by SHA256, and exposes file APIs for upload, download, listing, and image resize.

## Configuration

Storage config is loaded by Viper with priority: `.toml` > `.yaml` > `.env` > OS environment variables.

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

### Settings Table Entries

| Key | Default | Description |
|-----|---------|-------------|
| `storage.max_versions` | `5` | Max file versions per attachment |
| `storage.thumbnail.width` | `300` | Default thumbnail width |
| `storage.thumbnail.height` | `300` | Default thumbnail height |
| `storage.thumbnail.quality` | `85` | JPEG quality (1-100) |

## Storage Drivers

### Local

- Stores files under `[storage.local].path`
- Uses `[storage.local].base_url` for public URL generation
- Private files are served through a handler so auth can run per request

### S3

- Stores files in `[storage.s3].bucket`
- Supports AWS S3 and compatible endpoints via `endpoint`
- `use_path_style = true` supports MinIO-style path addressing
- Private file access uses signed URLs with configurable expiry

## Attachments Table

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
| `size` | BIGINT | NO | `0` | File size in bytes |
| `mime_type` | VARCHAR(255) | NO | — | MIME type |
| `ext` | VARCHAR(20) | NO | — | File extension |
| `hash` | VARCHAR(64) | NO | — | SHA256 checksum |
| `is_public` | BOOLEAN | NO | `false` | Public access flag |
| `version` | INTEGER | NO | `1` | Version number |
| `parent_id` | UUID | YES | NULL | FK → attachments. Original file |
| `thumbnail_path` | VARCHAR(1000) | YES | NULL | Thumbnail storage path |
| `metadata` | JSON | YES | NULL | Extra data such as width, height, duration |
| `active` | BOOLEAN | NO | `true` | Soft delete flag |
| `created_at` | DATETIME | NO | now | Upload timestamp |
| `updated_at` | DATETIME | NO | now | Last update |

### Indexes

- `idx_attachments_model_record` on (`model`, `record_id`, `field_name`)
- `idx_attachments_hash` on (`hash`)
- `idx_attachments_tenant` on (`tenant_id`)
- `idx_attachments_parent` on (`parent_id`)
- `idx_attachments_user` on (`user_id`)

## Template Variables

`path_format` and `name_format` support these variables.

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
| `{original}` | Original filename without extension, sanitized | `invoice_april` |
| `{ext}` | Original extension | `.pdf` |
| `{input.xxx}` | Form input field value | `{input.customer_name}` → `acme_corp` |
| `{data.xxx}` | Model record field value | `{data.invoice_number}` → `INV-2026-001` |

### Resolution

1. Parse `{...}` tokens from the template string
2. Resolve standard variables from session and current time
3. Resolve `{input.xxx}` from form submission data
4. Resolve `{data.xxx}` from the linked model record
5. Sanitize all resolved values
6. Replace unresolved variables with an empty string and log a warning

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/upload` | Yes | Single file upload |
| `POST` | `/api/uploads` | Yes | Multiple file upload |
| `GET` | `/api/files/:id` | Yes* | File metadata |
| `GET` | `/api/files/:id/download` | Yes* | Download file |
| `GET` | `/api/files/:id/thumbnail` | Yes* | Get thumbnail |
| `GET` | `/api/files/:id/resize` | Yes* | On-demand resize with `?w=200&h=200` |
| `GET` | `/api/files` | Yes | List files filtered by `model`, `record_id`, `field_name` |
| `DELETE` | `/api/files/:id` | Yes | Soft delete |
| `GET` | `/api/files/:id/versions` | Yes | List versions |

`Yes*` means auth is required unless `is_public = true`.

### 1. POST `/api/upload`

Single file upload.

```http
POST /api/upload
Content-Type: multipart/form-data

file: (binary)
model: invoice
record_id: uuid
field_name: attachments
is_public: false
metadata: {}
```

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

### 2. POST `/api/uploads`

Multiple file upload.

```http
POST /api/uploads
Content-Type: multipart/form-data

files: (binary[])
model: invoice
record_id: uuid
field_name: attachments
```

```json
{
  "files": [
    {
      "id": "uuid-1",
      "name": "invoice_april.pdf",
      "url": "/api/files/uuid-1/download",
      "size": 1048576,
      "mime_type": "application/pdf",
      "ext": ".pdf",
      "hash": "sha256...",
      "version": 1,
      "thumbnail_url": null,
      "created_at": "2026-04-24T15:30:45Z"
    }
  ],
  "total": 1,
  "errors": []
}
```

### 3. GET `/api/files/:id`

Returns file metadata.

```json
{
  "id": "uuid",
  "model": "invoice",
  "record_id": "record-uuid",
  "field_name": "attachments",
  "name": "invoice_april.pdf",
  "url": "/api/files/uuid/download",
  "storage": "local",
  "size": 1048576,
  "mime_type": "application/pdf",
  "ext": ".pdf",
  "hash": "sha256...",
  "is_public": false,
  "version": 1,
  "thumbnail_url": null,
  "metadata": {},
  "created_at": "2026-04-24T15:30:45Z",
  "updated_at": "2026-04-24T15:30:45Z"
}
```

### 4. GET `/api/files/:id/download`

Downloads the original file.

```http
GET /api/files/uuid/download
```

Response: file stream or redirect/signed URL depending on storage driver.

### 5. GET `/api/files/:id/thumbnail`

Returns the generated thumbnail for image uploads.

```http
GET /api/files/uuid/thumbnail
```

Response: thumbnail image stream.

### 6. GET `/api/files/:id/resize?w=200&h=200`

Generates and returns a resized image variant.

```http
GET /api/files/uuid/resize?w=200&h=200
```

Response: resized image stream.

### 7. GET `/api/files`

Lists files by upload context.

```http
GET /api/files?model=invoice&record_id=uuid&field_name=attachments
```

```json
{
  "files": [
    {
      "id": "uuid",
      "name": "invoice_april.pdf",
      "url": "/api/files/uuid/download",
      "version": 1,
      "size": 1048576,
      "mime_type": "application/pdf"
    }
  ],
  "total": 1
}
```

### 8. DELETE `/api/files/:id`

Soft-deletes the attachment.

```http
DELETE /api/files/uuid
```

```json
{
  "success": true
}
```

### 9. GET `/api/files/:id/versions`

Lists stored versions for a file.

```http
GET /api/files/uuid/versions
```

```json
{
  "files": [
    {
      "id": "uuid-v1",
      "version": 1,
      "name": "invoice_april.pdf",
      "created_at": "2026-04-24T15:30:45Z"
    },
    {
      "id": "uuid-v2",
      "version": 2,
      "name": "invoice_april_revised.pdf",
      "created_at": "2026-04-24T16:05:10Z"
    }
  ],
  "total": 2
}
```

## File Versioning

When a file is uploaded for an existing `model` + `record_id` + `field_name` + `parent_id` combination:

1. Find the latest version for that file chain
2. Set the new version to `latest + 1`
3. If version count exceeds `storage.max_versions`, soft-delete the oldest version
4. Keep older versions available through `GET /api/files/:id/versions`

## Thumbnail Generation

For image uploads (`image/*`):

1. Read the uploaded image
2. Resize it to the configured thumbnail dimensions while preserving aspect ratio
3. Save it as `{path}_thumb{ext}`
4. Store the result in `thumbnail_path`

### On-Demand Resize

`GET /api/files/:id/resize?w=200&h=200`

1. Validate dimensions with a maximum of `2000x2000`
2. Generate a resized image variant
3. Cache it as `{path}_w200_h200{ext}`
4. Return the generated image

## Duplicate Detection

On upload:

1. Compute SHA256 for the file
2. Query active attachments with the same `hash`, `model`, `record_id`, and `field_name`
3. If a match exists, return the existing attachment without re-uploading
4. Otherwise continue with normal upload flow

## Security

- **Private files** are the default and require auth
- **Public files** set `is_public = true` and skip auth checks
- **Local storage** serves private files through a handler, not a static directory
- **S3 storage** uses signed URLs with configurable expiry for private access
- **Validation** checks file size, allowed extensions, and MIME type
- **Sanitization** strips path traversal and unsafe characters from filenames
- **Authorization** checks user permission on the linked model for private files

## Scan Hook

The upload flow exposes a scan hook before storage write:

```go
type ScanHook interface {
    BeforePut(ctx context.Context, filename string, reader io.Reader) (io.Reader, error)
}
```

The hook point exists for integrations such as ClamAV or external scanning services.

## Orphan Cleanup

A cron agent removes orphaned uploads on a configurable interval, default `24h`:

1. Query active attachments where `record_id IS NULL` and `created_at` is older than 24 hours
2. Delete the file from storage
3. Hard-delete the attachment row

## Per-Model File Config

Models can override storage limits with `file_config`.

```json
{
  "name": "invoice",
  "file_config": {
    "max_size": 5242880,
    "allowed_extensions": [".pdf", ".jpg", ".png"]
  }
}
```

Supported overrides:

| Option | Description |
|--------|-------------|
| `max_size` | Per-model max upload size in bytes |
| `allowed_extensions` | Per-model extension whitelist |

## Stencil Component: `bc-field-file`

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `name` | string | — | Field name |
| `value` | string/string[] | — | Current file ID or file IDs |
| `multiple` | boolean | `false` | Allow multiple files |
| `accept` | string | — | Accepted MIME types |
| `max-size` | number | — | Max file size in bytes |
| `disabled` | boolean | `false` | Disable upload |
| `api-base` | string | `/api` | API base URL |

### Features

- Drag and drop zone
- Click to browse
- Upload progress bar
- Image preview via thumbnail
- File icon for non-image uploads
- File list with remove button
- Error display for size and type validation
