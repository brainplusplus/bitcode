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

- `base` is always installed first
- Circular dependencies are detected and rejected

## Installation Order

```
base → crm → sales
```

The engine scans the `modules/` directory, resolves dependencies, and installs in the correct order.

## CLI Commands

```bash
bitcode module list              # List available modules
bitcode module create my-module  # Scaffold new module
```

## Built-in Base Module

Always installed. Provides: user, role, group, permission, record_rule, audit_log, setting models + auth API + user API.
