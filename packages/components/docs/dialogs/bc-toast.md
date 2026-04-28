# bc-toast

> Toast notification with auto-dismiss

## Quick Start

```html
<bc-toast></bc-toast>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| open | boolean | false | Show state |
| dialog-title | string | '' | Toast title |
| message | string | '' | Toast message |
| variant | string | 'info' | success, error, warning, info |
| duration | number | 4000 | Auto-dismiss ms (0=manual) |
| position | string | 'top-right' | Position on screen |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcDialogClose | {type} | Dismissed |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| show(message?, variant?) | Promise<void> | Show toast |
| dismiss() | Promise<void> | Dismiss toast |

See [theming](../theming.md).

