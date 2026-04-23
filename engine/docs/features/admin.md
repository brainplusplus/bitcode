# Admin UI

Built-in web admin panel with Frappe-inspired sidebar layout for inspecting and managing the engine.

## Access

Open `http://localhost:8080/admin` in your browser.

## Layout

- **Sidebar** (left, fixed): Dashboard, Modules, Models, Views, Health links
- **Main content** (right): Breadcrumb + page content in card containers
- **Design**: Clean, modern — inspired by Frappe Framework admin

## Pages

### Dashboard (`/admin`)

Overview with:
- Stat cards: model count, module count, view count
- Installed modules table (name, version, category, status)
- Registered models table grouped by module (name, module badge, fields, data link)

### Models (`/admin/models`)

DocType-style list with:
- Module filter pills (All, base, crm, hrm)
- Count indicator (e.g., "12 of 16")
- Table: name, module, label, fields, inherit

### Model Detail (`/admin/models/:name`)

Three tabs (like Frappe DocType):

**Form tab** — Visual form preview:
- Each field rendered as a form input (text, toggle, select, link, file, date, color, rating, password)
- One2many/many2many shown as child table placeholders
- Link to view data

**Fields tab** — Field table:
- Numbered rows: name, type (code badge), label, required (green dot), unique (blue dot), default, relation (linked)
- Record rules table (groups + domain)
- Indexes table

**Connections tab** — Relationship map:
- Outgoing references (this model → other models via many2one/one2many/many2many)
- Incoming references (other models → this model, computed from all registered models)
- Related views (views that use this model)

### Model Data (`/admin/models/:name/data`)

Browse database records:
- Table view of all records (up to 50)
- Shows all non-virtual fields (excludes one2many, many2many, computed)
- Empty state for tables with no records

### Modules (`/admin/modules`)

List all installed modules:
- Name (linked to detail), version, label, category, dependencies, status

### Module Detail (`/admin/modules/:name`)

Three tabs:

**Overview tab**:
- Module info card (name, version, label, category, dependencies)
- Resources card (count of model/api/process/view/template/script/data/i18n patterns)
- Models in this module (linked to model detail)
- Views for this module

**Permissions tab**:
- Permissions table (key + description)
- Groups table (key, label, implies hierarchy)

**Menu tab**:
- Menu structure visualization (groups with children and view links)
- Settings table (key, type, default value)

### Views (`/admin/views`)

List all registered views with module filter:
- Name (linked to detail), type badge, model, title, module, editable status (embedded/editable)

### View Detail (`/admin/views/:module/:name`)

Four tabs:

**Info tab**:
- Metadata card (name, type, model, title, module, file path, editable status)
- Quick actions (open view, edit JSON, view model, revisions)

**Preview tab**:
- Iframe rendering the view without layout
- Link to open full view in app

**Editor tab** — split mode (Visual / JSON / Split):
- Visual: `bc-view-editor` Stencil component (drag-drop field palette, canvas with sections/rows, properties panel)
- JSON: textarea with syntax highlighting
- Bi-directional sync between visual and JSON
- Save button (disabled for embedded views)
- Publish button for embedded views (extracts to project)

**Revisions tab**:
- Revision history table (version, created at, created by)
- Rollback button per revision
- Revisions auto-created on every save, configurable limit via `view_revision_limit` setting (default: 50)

### Admin API (`/admin/api/`)

JSON API endpoints for programmatic access:

| Method | Path | Description |
|--------|------|-------------|
| GET | /admin/api/views/:module/:name | View detail + revision list |
| GET | /admin/api/views/:module/:name/json | Raw JSON content |
| POST | /admin/api/views/:module/:name | Save JSON (validate + write file + create revision) |
| POST | /admin/api/views/:module/:name/rollback/:version | Restore revision |
| GET | /admin/api/views/:module/:name/preview | Render view HTML without layout |
| POST | /admin/api/views/:module/:name/publish | Extract embedded view to project |

### Health (`/admin/health`)

System health dashboard:
- Status indicator (green pulsing dot + "Healthy")
- Stat cards: models, modules, views, processes count
- Infrastructure card: engine, version, database driver, cache driver
- Installed modules with version and status
- Registered processes as badge pills
- Link to JSON API (`/health`)

## No Authentication Required

The admin panel is currently open. In production, add authentication middleware to the `/admin` routes.
