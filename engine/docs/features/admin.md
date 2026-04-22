# Admin UI

Built-in web admin panel for inspecting and managing the engine.

## Access

Open `http://localhost:8080/admin` in your browser.

## Pages

### Dashboard (`/admin`)

Overview with:
- Model count, module count, view count
- Installed modules table (name, version, status)
- Registered models table (name, module, field count, data link)

### Models (`/admin/models`)

List all registered models with:
- Name, module, label, field count, inheritance, record rule count

### Model Detail (`/admin/models/:name`)

Inspect a specific model:
- All fields with type, required, unique, default, FK model
- Record rules with groups and domain filters
- Link to view data

### Model Data (`/admin/models/:name/data`)

Browse actual database records:
- Table view of all records (up to 50)
- Shows all non-virtual fields (excludes one2many, many2many, computed)

### Modules (`/admin/modules`)

List all installed modules:
- Name, version, label, category, dependencies, status

### Views (`/admin/views`)

List all registered views:
- Name, type (list/form/kanban/etc), model, title
- Preview link to open the SSR view

## No Authentication Required

The admin panel is currently open. In production, you should add authentication middleware to the `/admin` routes.
