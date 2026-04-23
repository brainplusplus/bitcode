# View Editor & Versioning System Design

**Date**: 2026-04-23
**Status**: Approved

## Problem

The admin panel's Views page is empty (ViewResolver called before modules load). Views need:
1. A proper list + detail page in admin
2. Preview rendering (without layout)
3. JSON editor with validation + syntax highlighting
4. Visual drag-drop editor (Stencil Web Component)
5. Revision history stored in database with rollback
6. Embedded views are read-only until published

## Architecture

```
Source of truth: JSON files on disk
DB table: view_revisions (revision history / snapshots)
Admin UI: list -> detail (info + preview + editor + revisions)
Save flow: editor -> POST API -> validate -> write file -> create revision -> hot-reload
Visual editor: bc-view-editor Stencil Web Component
```

## Data Model

### `view_revisions` Table

Auto-migrated by engine (internal table, not a module model):

| Column | Type | Description |
|--------|------|-------------|
| id | UUID (TEXT) | Primary key |
| view_key | TEXT | View identifier (e.g., "crm/contact_list") |
| version | INTEGER | Auto-increment per view_key |
| content | TEXT | Full JSON snapshot |
| created_by | TEXT | Username |
| created_at | TIMESTAMP | Creation time |
| is_active | BOOLEAN | Currently active version |

Indexes: `(view_key)`, `(view_key, version)`

### Settings

- `view_revision_limit`: 0 = unlimited, N = keep last N per view. Default: 50.
- Configurable in admin settings page.

## Admin API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /admin/api/views | List all views with metadata |
| GET | /admin/api/views/:key | View detail + current JSON + revision list |
| GET | /admin/api/views/:key/json | Raw JSON content |
| POST | /admin/api/views/:key | Save JSON -> validate -> write file -> create revision |
| POST | /admin/api/views/:key/rollback/:version | Restore revision -> write file |
| GET | /admin/api/views/:key/preview | Render view HTML without layout (iframe) |
| POST | /admin/api/views/:key/publish | Extract embedded view to project (make editable) |

### Save Flow

```
POST /admin/api/views/crm/contact_form
Body: { "content": "{...json...}" }

1. Validate JSON (parser.ParseView)
2. Check editable (not embedded temp dir)
3. Write to file on disk
4. Insert revision in DB
5. Cleanup old revisions if over limit
6. Reload view in engine
7. Return { "ok": true, "version": N }
```

### Rollback Flow

```
POST /admin/api/views/crm/contact_form/rollback/3

1. Get revision #3 from DB
2. Write content to file
3. Insert new revision (version N+1, content from #3)
4. Reload view
5. Return { "ok": true, "version": N+1, "restored_from": 3 }
```

## Admin UI Pages

### Views List (/admin/views)

- Module filter pills (All, base, crm, hrm)
- Count indicator
- Table: Name (linked), Type badge, Model, Title, Module, Editable status

### View Detail (/admin/views/:key) - 4 Tabs

**Info tab**:
- Metadata card (name, type, model, title, module)
- Editable status badge
- If embedded: "Publish to edit" button
- File path

**Preview tab**:
- Iframe loading /admin/api/views/:key/preview
- Refresh button

**Editor tab** - split pane:
- Left: bc-view-editor Stencil component (visual drag-drop)
- Right: JSON textarea with validation + highlighting
- Mode toggle: Visual / JSON / Split
- Save button (disabled if embedded)
- Bi-directional sync

**Revisions tab**:
- Table: Version, Created At, Created By
- Preview + Rollback buttons per row
- Active version highlighted

### Embedded Protection

- Embedded views: editor read-only, save disabled
- Badge: "Embedded - publish to edit"
- Publish button extracts to project

## bc-view-editor Stencil Component

Location: `packages/components/src/components/view-editor/`

### Props/Events

```typescript
@Prop() viewJson: string;           // Current view JSON
@Prop() modelFields: FieldDef[];    // Available fields from model
@Prop() readonly: boolean;          // Disable editing

@Event() viewChanged: EventEmitter<{ json: string }>;
```

### Layout

```
+------------------+------------------------+------------------+
| Field Palette    | Canvas                 | Properties       |
|                  |                        |                  |
| [drag fields]   | [sections]             | [selected item]  |
|                  |   [rows]               |   width: [6]     |
|                  |     [fields]           |   widget: [...]  |
|                  |   [+ Add Row]          |   readonly: [ ]  |
|                  | [tabs]                 |                  |
|                  | [+ Add Section]        |                  |
+------------------+------------------------+------------------+
```

### Features

- Drag fields from palette to rows
- Drag rows to reorder within section
- Drag sections to reorder
- Resize field width (1-12 grid dropdown)
- Property panel for selected item
- Add/remove sections, rows, tabs
- JSON sync: parse viewJson -> internal tree -> emit viewChanged

### Tech

- Pure Stencil.js (TypeScript + CSS)
- HTML5 Drag and Drop API
- CSS Grid for layout preview
- Files: editor.tsx, canvas.tsx, palette.tsx, properties.tsx, types.ts

## Files Changed

### Engine

| File | Change |
|------|--------|
| internal/infrastructure/persistence/view_revision.go | NEW - ViewRevisionRepository |
| internal/presentation/admin/admin.go | MODIFY - fix ViewResolver, view list/detail pages |
| internal/presentation/admin/admin_api.go | NEW - JSON API handlers |
| internal/app.go | MODIFY - auto-migrate view_revisions, wire API |
| embedded/modules/base/module.json | MODIFY - add view_revision_limit setting |

### Stencil Components

| File | Change |
|------|--------|
| packages/components/src/components/view-editor/bc-view-editor.tsx | NEW |
| packages/components/src/components/view-editor/bc-view-editor.css | NEW |
| packages/components/src/components/view-editor/canvas.tsx | NEW |
| packages/components/src/components/view-editor/palette.tsx | NEW |
| packages/components/src/components/view-editor/properties.tsx | NEW |
| packages/components/src/components/view-editor/types.ts | NEW |

## Execution Order

1. Fix ViewResolver (lazy function - already started)
2. view_revision.go - DB table + repository
3. Admin API endpoints (admin_api.go)
4. Admin UI pages (views list + detail with tabs)
5. bc-view-editor Stencil component
6. Wire everything together
7. Tests
