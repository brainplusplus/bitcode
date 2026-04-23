# Notepad

## Priority Context

- Working on View Editor & Versioning feature. Design doc: `engine/docs/plans/2026-04-23-view-editor-versioning-design.md`
- AdminPanel.viewDefs changed from static map to lazy ViewResolver function (partially done - need to also expose ViewInfo with file path + editable status)
- admin.go has partial edit: `ViewInfo` struct added, `ViewResolver` type changed to `func() map[string]*ViewInfo` but app.go `viewsByName` still returns old type - NEEDS FIXING before implementation
- 175 tests passing, both binaries build OK

## Working Memory

### 2026-04-23 — View Editor Design Approved
- view_revisions DB table for revision history
- Admin API: GET/POST views, rollback, preview, publish
- Admin UI: views list + detail (4 tabs: info, preview, editor, revisions)
- bc-view-editor Stencil component: drag-drop canvas, field palette, property panel
- File = source of truth, DB = revisions/drafts
- Embedded views = read-only until published
- Settings: view_revision_limit (default 50)
