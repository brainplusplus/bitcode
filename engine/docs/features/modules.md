# Modules

Modules are the unit of deployment. Everything lives in a module.

## Module Definition

```json
{
  "name": "sales",
  "version": "1.0.0",
  "label": "Sales Management",
  "depends": ["base", "crm"],
  "category": "Sales",
  "models": ["models/*.json"],
  "apis": ["apis/*.json"],
  "processes": ["processes/*.json"],
  "agents": ["agents/*.json"],
  "views": ["views/*.json"],
  "templates": ["templates/*.html"],
  "scripts": ["scripts/*"],
  "data": ["data/*.json"],
  "i18n": ["i18n/*.json"],
  "menu_visibility": "app",
  "include_menus": [
    {"module": "base", "views": ["user_list"]}
  ],
  "permissions": {
    "sales.order.read": "Read orders",
    "sales.order.create": "Create orders"
  },
  "groups": {
    "sales.user":    { "label": "Sales / User",    "implies": ["base.user"] },
    "sales.manager": { "label": "Sales / Manager", "implies": ["sales.user"] }
  },
  "menu": [
    { "label": "Sales", "icon": "shopping-cart", "children": [
      { "label": "Orders", "view": "order_list" }
    ]}
  ],
  "settings": {
    "default_currency": { "type": "string", "default": "USD" }
  }
}
```

## Module Structure

```
modules/sales/
├── module.json          # Module definition
├── models/              # Model definitions
├── apis/                # API definitions
├── processes/           # Process + workflow definitions
├── agents/              # Agent definitions
├── views/               # View definitions
├── templates/           # HTML templates
├── scripts/             # TypeScript/Python plugins
├── data/                # Demo/seed data
└── i18n/                # Translations
```

## Dependencies

Modules can depend on other modules. The engine resolves dependencies automatically using topological sort.

```json
"depends": ["base", "crm"]
```

- `base` is always available (embedded in binary)
- Circular dependencies are detected and rejected

## 3-Layer Module Resolution

The engine resolves module files from 3 layers, highest priority first:

```
Layer 1: Project    → ./modules/base/models/user.json     (user override)
Layer 2: Global     → ~/.bitcode/modules/base/...          (shared across projects)
Layer 3: Embedded   → [binary]/base/...                    (default fallback)
```

Resolution is **per-file**. If `./modules/base/models/user.json` exists in the project, it overrides the embedded version. All other base files fall back to embedded.

### How It Works

- Base module is embedded in the engine binary via Go `embed.FS`
- `ModuleFS` interface abstracts file access (DiskFS, EmbedFS, LayeredFS)
- `LayeredFS` composes multiple FS layers with priority-based per-file resolution
- Engine discovers modules across all layers, deduplicates, resolves dependencies

## Publish Command

Extract embedded module files to your project for customization. Similar to Laravel's `artisan vendor:publish`.

```bash
bitcode publish base                    # Publish entire base module
bitcode publish base --models           # Publish only models
bitcode publish base --views            # Publish only views
bitcode publish base --templates        # Publish only templates
bitcode publish base models/user.json   # Publish single file
bitcode publish base --force            # Overwrite existing files
bitcode publish base --dry-run          # Preview without writing
bitcode publish --list                  # List publishable modules
```

Default behavior: skip existing files (no overwrite). Use `--force` to overwrite.

## Menu Visibility

Control where a module's menu appears:

```json
"menu_visibility": "app"
```

| Value | Behavior |
|-------|----------|
| `"app"` | Menu appears in app views sidebar (default) |
| `"admin"` | Menu only appears in admin panel |

The base module uses `"menu_visibility": "admin"` — its menu (Users, Roles, Permissions, Settings) only appears in the admin panel, not in the app sidebar.

## Include Menus

Import menu items from other modules into your module's sidebar:

```json
"include_menus": [
  {"module": "base", "views": ["user_list", "role_list"]},
  {"module": "hrm", "views": ["employee_list"]}
]
```

- `views` — array of specific view names to include. Omit to include all.
- Included items are merged into matching menu groups of the importing module.
- Use case: CRM module wants to show "Users" link from base module in its sidebar.

## CLI Commands

```bash
bitcode module list              # List available modules
bitcode module create my-module  # Scaffold new module
bitcode publish base             # Extract embedded module to project
bitcode publish --list           # List publishable modules
```

## Built-in Base Module

Embedded in the engine binary. Always available. Provides: user, role, group, permission, record_rule, audit_log, setting models + auth API + default templates.

Config: `GLOBAL_MODULE_DIR` env var sets the global module directory (default: `~/.bitcode/modules`).

## View Versioning

Views saved from the admin editor are versioned in the `view_revisions` database table. Each save creates a new revision. Rollback restores a previous revision's content to the file.

Settings:
- `view_revision_limit`: Maximum revisions to keep per view. `0` = unlimited, default: `50`.
