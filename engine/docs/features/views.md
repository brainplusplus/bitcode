# Views & Templates

Views define UI pages. Templates provide custom HTML rendering. All views are automatically wrapped in a layout template with sidebar navigation, navbar, and modern styling.

## View Types

### List View

```json
{
  "name": "order_list",
  "type": "list",
  "model": "order",
  "title": "Sales Orders",
  "fields": ["order_date", "customer_id", "total", "status"],
  "filters": ["status", "order_date"],
  "sort": { "field": "order_date", "order": "desc" },
  "actions": [
    { "label": "Confirm", "process": "confirm_order", "permission": "order.confirm", "visible": "status == 'draft'" }
  ]
}
```

### Form View

```json
{
  "name": "order_form",
  "type": "form",
  "model": "order",
  "title": "Sales Order",
  "layout": [
    { "row": [
      { "field": "customer_id", "width": 6 },
      { "field": "order_date", "width": 3 },
      { "field": "status", "width": 3, "readonly": true }
    ]},
    { "tabs": [
      { "label": "Lines", "view": "order_line_list" },
      { "label": "Notes", "fields": ["notes"] }
    ]}
  ],
  "actions": [
    { "label": "Confirm", "process": "confirm_order", "variant": "primary", "visible": "status == 'draft'" }
  ]
}
```

### Kanban View

```json
{
  "name": "lead_kanban",
  "type": "kanban",
  "model": "lead",
  "title": "Lead Pipeline",
  "group_by": "status",
  "fields": ["name", "company", "expected_revenue"]
}
```

### Custom View (Template)

```json
{
  "name": "dashboard",
  "type": "custom",
  "template": "templates/dashboard.html",
  "data_sources": {
    "orders": { "model": "order", "domain": [["status", "=", "confirmed"]] },
    "stats": { "process": "get_stats" }
  }
}
```

## Layout System

All views rendered via `/app/:module/*` are automatically wrapped in a layout template. The layout provides:

- **Sidebar** — Module navigation with menu items from `module.json`
- **Navbar** — Breadcrumbs, user info, logout
- **Content area** — Where the view renders

### How It Works

1. The view content (list, form, kanban, etc.) is rendered first
2. The result is injected into `layout.html` as `{{.Content}}`
3. Layout also receives `{{.Title}}`, `{{.Module}}`, `{{.Username}}`, `{{.Menu}}`

### Default Templates (Base Module)

```
modules/base/templates/
├── layout.html                    # Main layout (sidebar + navbar + content)
├── partials/
│   ├── sidebar.html               # Left sidebar navigation
│   ├── navbar.html                # Top navigation bar
│   ├── pagination.html            # Reusable pagination component
│   ├── status_badge.html          # Status badge component
│   └── actions.html               # Action buttons component
└── views/
    ├── list.html                  # List/table view template
    ├── form.html                  # Form view template
    ├── kanban.html                # Kanban board template
    ├── calendar.html              # Calendar view template
    ├── chart.html                 # Chart view template
    ├── login.html                 # Login page (standalone)
    └── home.html                  # Home dashboard
```

### Overriding Templates

Modules can override default templates by providing their own in `templates/`. Templates are loaded in module dependency order, so later modules override earlier ones.

### Partials

Partials are shared across all templates. Any file in a `partials/` subdirectory is automatically available:

```html
{{template "templates/partials/status_badge.html" .status}}
{{template "templates/partials/pagination.html" .}}
```

## Cross-Module Views

### Registering Views in Other Modules

A view can register itself in another module's namespace using `register_to`:

```json
{
  "name": "customer_widget",
  "type": "list",
  "model": "customer",
  "register_to": ["crm", "sales"]
}
```

If the target module is not installed, the registration is silently skipped (no error).

### Referencing Cross-Module Views

Use `module.view_name` syntax in URLs:

```
/app/crm/sales.order_list    → renders sales module's order_list view
```

## Templates

Templates use Go `html/template` syntax.

```html
<div class="dashboard">
  <h1>Dashboard</h1>
  {{range .orders}}
    <div>{{.name}} - {{formatCurrency .total}}</div>
  {{end}}
</div>
```

### Built-in Helpers

| Helper | Usage | Description |
|--------|-------|-------------|
| `formatDate` | `{{formatDate .date}}` | Format as YYYY-MM-DD |
| `formatDateTime` | `{{formatDateTime .date}}` | Format as YYYY-MM-DD HH:MM |
| `formatCurrency` | `{{formatCurrency .amount}}` | Format as $0.00 |
| `truncate` | `{{truncate .text 50}}` | Truncate with ... |
| `upper` | `{{upper .text}}` | Uppercase |
| `lower` | `{{lower .text}}` | Lowercase |
| `title` | `{{title .text}}` | Title case |
| `dict` | `{{dict "key" "value"}}` | Create map |
| `eq` | `{{if eq .status "active"}}` | Equality check |
| `neq` | `{{if neq .status "draft"}}` | Inequality check |
| `safeHTML` | `{{safeHTML .html}}` | Render raw HTML |
| `add` | `{{add .page 1}}` | Integer addition |
| `sub` | `{{sub .page 1}}` | Integer subtraction |
| `seq` | `{{range seq 1 10}}` | Generate integer sequence |
| `join` | `{{join .list ", "}}` | Join strings |
| `contains` | `{{if contains .text "foo"}}` | String contains |
| `hasPrefix` | `{{if hasPrefix .url "http"}}` | String prefix check |
| `default` | `{{default "N/A" .value}}` | Default value if empty |

### SSR Route

Views are accessible at `GET /app/{module}/{view_name}`:

```bash
curl http://localhost:8080/app/crm/lead_list
curl http://localhost:8080/app/sales/order_form
```

## Action Options

| Option | Type | Description |
|--------|------|-------------|
| `label` | string | Button text |
| `process` | string | Process to execute |
| `permission` | string | Required permission |
| `variant` | string | Button style: primary, danger |
| `visible` | string | Condition expression |
| `confirm` | string | Confirmation dialog text |

## View Options

| Option | Type | Description |
|--------|------|-------------|
| `register_to` | string[] | Register this view in other modules (graceful if missing) |
