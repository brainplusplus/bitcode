# View Editor & Versioning — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add view list/detail pages to admin with preview, JSON editor, visual drag-drop editor (Stencil), and revision history stored in database.

**Architecture:** File = source of truth. DB stores revision snapshots. Admin API handles save (validate → write file → create revision). Stencil `bc-view-editor` component for visual editing. 3-layer resolution determines if view is editable (embedded = read-only).

**Tech Stack:** Go 1.23+, Fiber v2, GORM, Stencil.js (TypeScript), HTML5 Drag and Drop API

**Design Doc:** `docs/plans/2026-04-23-view-editor-versioning-design.md`

---

## Task 1: Fix ViewResolver — Lazy Function

**Files:**
- Modify: `engine/internal/presentation/admin/admin.go`
- Modify: `engine/internal/app.go`

**Goal:** AdminPanel gets views lazily (called at request time, not at setup time) so views are populated after LoadModules().

**Steps:**
1. In `admin.go`: `ViewInfo` struct already added. Update `ViewResolver` type to return `map[string]*ViewInfo`.
2. In `app.go`: Update `viewsByName()` to return `map[string]*admin.ViewInfo` including Module, FilePath, Editable fields.
3. Editable logic: if file path starts with os.TempDir() prefix → embedded (not editable). Otherwise editable.
4. Update all admin.go handlers that use `a.views()` to work with `*ViewInfo` (access `.Def` for the ViewDefinition).
5. Build: `go build ./cmd/engine/`
6. Test: `go test ./... -count=1`

---

## Task 2: ViewRevision Repository

**Files:**
- Create: `engine/internal/infrastructure/persistence/view_revision.go`
- Create: `engine/internal/infrastructure/persistence/view_revision_test.go`

**Goal:** DB table `view_revisions` + repository for CRUD + cleanup.

**Steps:**
1. Define `ViewRevision` GORM model struct (ID, ViewKey, Version, Content, CreatedBy, CreatedAt, IsActive).
2. `ViewRevisionRepository` with methods:
   - `AutoMigrate(db)` — create table
   - `Create(revision)` — insert + auto-increment version per view_key
   - `ListByViewKey(viewKey, limit)` — list revisions ordered by version desc
   - `GetByVersion(viewKey, version)` — get specific revision
   - `GetLatest(viewKey)` — get latest revision
   - `Cleanup(viewKey, keepN)` — delete old revisions beyond limit
3. Write tests using SQLite in-memory DB.
4. Build + test.

---

## Task 3: Auto-Migrate view_revisions in app.go

**Files:**
- Modify: `engine/internal/app.go`

**Goal:** Engine auto-creates `view_revisions` table on startup.

**Steps:**
1. In `NewApp()`, after DB connection, call `persistence.AutoMigrateViewRevisions(db)`.
2. Build + test.

---

## Task 4: Admin API Endpoints

**Files:**
- Create: `engine/internal/presentation/admin/admin_api.go`
- Modify: `engine/internal/presentation/admin/admin.go` (add routes)

**Goal:** JSON API for view CRUD, save, rollback, preview.

**Steps:**
1. Add `ViewRevisionRepo` field to `AdminPanel` struct. Update constructor + app.go caller.
2. Create `admin_api.go` with handlers:
   - `GET /admin/api/views` — list all views as JSON
   - `GET /admin/api/views/:key` — view detail + revisions
   - `GET /admin/api/views/:key/json` — raw JSON content (read file)
   - `POST /admin/api/views/:key` — save: validate → check editable → write file → create revision → return
   - `POST /admin/api/views/:key/rollback/:version` — restore revision → write file → create revision
   - `GET /admin/api/views/:key/preview` — render view HTML without layout (call ViewRenderer with UseLayout=false)
   - `POST /admin/api/views/:key/publish` — extract embedded view file to project modules dir
3. Register routes in `RegisterRoutes()`.
4. Build + test manually with curl.

---

## Task 5: Admin Views List Page

**Files:**
- Modify: `engine/internal/presentation/admin/admin.go`

**Goal:** Replace empty views list with proper list page (module filter, count, linked to detail).

**Steps:**
1. Rewrite `listViews()` handler:
   - Module filter pills (from view modules)
   - Count indicator
   - Table: Name (linked to /admin/views/:key), Type badge, Model, Title, Module badge, Editable status icon
2. Build + verify in browser.

---

## Task 6: Admin View Detail Page (Info + Preview + Revisions tabs)

**Files:**
- Modify: `engine/internal/presentation/admin/admin.go`

**Goal:** View detail page with tabs: Info, Preview, Editor, Revisions.

**Steps:**
1. Add route: `admin.Get("/views/:key", a.viewDetail)` — note: key contains "/" so use wildcard or encode.
   - Actually use `admin.Get("/views/:module/:name", a.viewDetail)` since key = "module/name".
2. `viewDetail()` handler with tab query param.
3. **Info tab**: metadata card (name, type, model, title, module, file path, editable badge).
4. **Preview tab**: iframe pointing to `/admin/api/views/:module/:name/preview`. Refresh button (JS reload iframe).
5. **Editor tab**: split pane placeholder — left: `<bc-view-editor>` component tag, right: textarea with JSON. Save button. Mode toggle buttons. Load Stencil component JS via script tag.
6. **Revisions tab**: fetch from `/admin/api/views/:module/:name` and render revision table. Rollback buttons with JS fetch POST.
7. Build + verify.

---

## Task 7: bc-view-editor Stencil Component — Types & Scaffold

**Files:**
- Create: `packages/components/src/components/view-editor/types.ts`
- Create: `packages/components/src/components/view-editor/bc-view-editor.tsx`
- Create: `packages/components/src/components/view-editor/bc-view-editor.css`

**Goal:** Scaffold the component with props/events, basic rendering.

**Steps:**
1. Define types: `EditorField`, `EditorRow`, `EditorSection`, `EditorLayout` (internal tree representation).
2. Create component with `@Prop() viewJson`, `@Prop() modelFields`, `@Prop() readonly`, `@Event() viewChanged`.
3. Parse viewJson → internal layout tree on prop change.
4. Render basic 3-panel layout (palette | canvas | properties) with placeholder content.
5. `npm run build` in packages/components/.

---

## Task 8: bc-view-editor — Field Palette

**Files:**
- Modify: `packages/components/src/components/view-editor/bc-view-editor.tsx`

**Goal:** Left panel showing available fields from model, draggable.

**Steps:**
1. Render `modelFields` as draggable items.
2. HTML5 drag: `draggable="true"`, `onDragStart` sets field data.
3. Visual: field name + type badge, hover highlight.
4. Filter/search input at top.
5. Build + verify.

---

## Task 9: bc-view-editor — Canvas (Drop Zones + Sections + Rows)

**Files:**
- Modify: `packages/components/src/components/view-editor/bc-view-editor.tsx`
- Modify: `packages/components/src/components/view-editor/bc-view-editor.css`

**Goal:** Center panel with droppable sections/rows, field rendering.

**Steps:**
1. Render sections from internal layout tree.
2. Each section: title + rows. Each row: fields with width (CSS grid 12-col).
3. Drop zones: `onDragOver` + `onDrop` on rows to accept fields from palette.
4. Add Section / Add Row buttons.
5. Remove section/row/field buttons (X icon).
6. Drag to reorder rows within section (dragstart/dragover/drop with index tracking).
7. On any change: rebuild JSON from tree → emit `viewChanged`.
8. Build + verify.

---

## Task 10: bc-view-editor — Properties Panel

**Files:**
- Modify: `packages/components/src/components/view-editor/bc-view-editor.tsx`

**Goal:** Right panel showing properties of selected field/section.

**Steps:**
1. Track `selectedItem` state (field, row, or section).
2. Click on canvas item → set selected.
3. Properties panel renders based on selected type:
   - Field: width (1-12 dropdown), widget (dropdown), readonly (checkbox), formula (input)
   - Section: title (input), collapsible (checkbox)
   - Tab: label (input)
4. On property change: update tree → emit viewChanged.
5. Build + verify.

---

## Task 11: bc-view-editor — JSON Sync

**Files:**
- Modify: `packages/components/src/components/view-editor/bc-view-editor.tsx`

**Goal:** Bi-directional sync between visual editor and JSON.

**Steps:**
1. `viewJson` prop change → parse → update internal tree → re-render canvas.
2. Canvas edit → update tree → serialize to JSON → emit `viewChanged`.
3. Handle parse errors gracefully (show error message, don't crash).
4. Build + verify.

---

## Task 12: Wire Editor into Admin Detail Page

**Files:**
- Modify: `engine/internal/presentation/admin/admin.go`
- Modify: `engine/internal/app.go` (serve Stencil component assets)

**Goal:** Admin view detail editor tab loads bc-view-editor component and wires save.

**Steps:**
1. Serve Stencil dist files from `/admin/assets/components/` (or use existing component asset route).
2. Editor tab HTML: load component script, render `<bc-view-editor>` with props.
3. Inline JS: listen to `viewChanged` event → update JSON textarea. Save button → POST to API.
4. Mode toggle: Visual / JSON / Split (show/hide panels via CSS).
5. Readonly mode for embedded views.
6. Build engine + build components + verify end-to-end.

---

## Task 13: Settings — view_revision_limit

**Files:**
- Modify: `engine/embedded/modules/base/module.json`

**Goal:** Add configurable revision limit setting.

**Steps:**
1. Add to base module.json settings: `"view_revision_limit": { "type": "integer", "default": 50 }`.
2. Admin API save handler reads this setting to determine cleanup limit.
3. Build + test.

---

## Task 14: Integration Testing + Final Verification

**Steps:**
1. Run full Go test suite: `go test ./... -count=1`
2. Build both binaries: `go build ./cmd/engine/ && go build ./cmd/bitcode/`
3. Build Stencil components: `cd packages/components && npm run build`
4. Manual smoke test:
   - Start engine with sample ERP
   - Open /admin/views — verify list shows views
   - Click a view — verify detail page with tabs
   - Preview tab — verify iframe renders view
   - Editor tab — verify JSON editor works, save works (for non-embedded)
   - Revisions tab — verify revision history after save
   - Rollback — verify restore works
5. Update docs.
