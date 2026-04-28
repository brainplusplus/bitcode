# bc-dialog-confirm

> Confirmation dialog with overlay

## Quick Start

```html
<bc-dialog-confirm></bc-dialog-confirm>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| open | boolean | false | Open state |
| dialog-title | string | '' | Dialog title |
| size | string | 'sm' | Dialog size (sm, md, lg) |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcDialogClose | {type} | Dialog closed |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| openDialog() | Promise<void> | Open |
| closeDialog() | Promise<void> | Close |

See [theming](../theming.md).

