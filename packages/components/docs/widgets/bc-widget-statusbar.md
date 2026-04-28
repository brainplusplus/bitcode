# bc-widget-statusbar

> Status bar showing workflow states

## Quick Start

```html
<bc-widget-statusbar></bc-widget-statusbar>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| states | string (JSON) | '[]' | Array of state names |
| value | string | '' | Current state |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| setValue(value) | Promise<void> | Set current state |
| getValue() | Promise<string> | Get current state |

See [theming](../theming.md).

