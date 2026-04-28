# bc-header

> Page header with action buttons and status bar

## Quick Start

```html
<bc-header></bc-header>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| buttons | string (JSON) | '[]' | Array of {label, process, variant} |
| status-field | string | '' | Status field name |
| status-value | string | '' | Current status |
| states | string (JSON) | '[]' | Array of state names |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcActionClick | {process} | Button clicked |

See [theming](../theming.md).

