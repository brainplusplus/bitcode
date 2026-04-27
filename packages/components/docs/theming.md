# Theming

All visual aspects are controlled via CSS custom properties (`--bc-*`). No Tailwind, no CSS-in-JS, no build step.

## Built-in Themes

| Theme | How to Apply |
|-------|-------------|
| Light | Default — no attribute needed |
| Dark | `data-bc-theme="dark"` on any ancestor |
| System | `BcSetup.configure({ theme: 'system' })` — auto-detects OS preference |

## Usage

### HTML Attribute (recommended)

```html
<body data-bc-theme="dark">
  <bc-field-string name="x" label="Dark field" />
</body>
```

### Scoped Theme

```html
<div data-bc-theme="dark">
  <bc-field-string name="x" label="Dark field" />
</div>
<bc-field-string name="y" label="Light field" />
```

### JavaScript

```javascript
BcSetup.configure({ theme: 'dark' });
BcSetup.configure({ theme: 'system' });  // auto-detect OS
BcSetup.configure({ theme: 'light' });
```

### Meta Tag

```html
<meta name="bc-theme" content="dark">
```

### Auto Dark (zero JS)

If no `data-bc-theme` attribute is set, the system automatically respects `prefers-color-scheme: dark` via CSS media query. No JavaScript needed.

## Custom Theme

Create a CSS file that overrides only the variables you need:

```css
[data-bc-theme="corporate"] {
  --bc-primary: #003366;
  --bc-primary-hover: #004488;
  --bc-font-family: 'Segoe UI', Tahoma, sans-serif;
  --bc-radius-sm: 0;
  --bc-radius-md: 0;
  --bc-radius-lg: 2px;
}
```

Apply:

```html
<link rel="stylesheet" href="corporate-theme.css">
<body data-bc-theme="corporate">
```

## Size Variants

Fields support `size` prop: `sm`, `md` (default), `lg`.

```html
<bc-field-string name="x" size="sm" />
<bc-field-string name="y" size="lg" />
```

CSS tokens per size:

| Token | SM | MD | LG |
|-------|----|----|-----|
| `--bc-input-height-{size}` | 1.875rem | 2.5rem | 3rem |
| `--bc-input-padding-x-{size}` | 0.5rem | 0.75rem | 1rem |
| `--bc-input-padding-y-{size}` | 0.25rem | 0.5rem | 0.625rem |
| `--bc-font-size-input-{size}` | 0.8125rem | 0.875rem | 1rem |
| `--bc-label-size-{size}` | 0.75rem | 0.875rem | 1rem |

Global default size via BcSetup:

```javascript
BcSetup.configure({ size: 'sm' });
```

## All CSS Variables

See `src/global/global.css` for the complete list. Key categories:

- **Colors**: `--bc-primary`, `--bc-success`, `--bc-warning`, `--bc-danger`, `--bc-info`
- **Background**: `--bc-bg`, `--bc-bg-secondary`, `--bc-bg-tertiary`
- **Text**: `--bc-text`, `--bc-text-secondary`, `--bc-text-placeholder`
- **Border**: `--bc-border-color`, `--bc-border-color-focus`
- **Typography**: `--bc-font-family`, `--bc-font-size-*`
- **Spacing**: `--bc-spacing-xs` through `--bc-spacing-2xl`
- **Radius**: `--bc-radius-sm` through `--bc-radius-full`
- **Shadows**: `--bc-shadow-sm` through `--bc-shadow-xl`
- **Input**: `--bc-input-height`, `--bc-input-bg`, `--bc-input-focus-ring`
- **Validation**: `--bc-field-valid-color`, `--bc-field-invalid-color`
- **Dropdown**: `--bc-dropdown-bg`, `--bc-dropdown-shadow`
- **Table**: `--bc-table-header-bg`, `--bc-table-row-hover`
- **Chart**: `--bc-chart-bg`, `--bc-chart-colors`
