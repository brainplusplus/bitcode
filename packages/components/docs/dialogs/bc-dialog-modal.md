# bc-dialog-modal

> Modal dialog with overlay

## Quick Start

```html
<bc-dialog-modal></bc-dialog-modal>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| open | boolean | false | Open state |
| dialog-title | string | '' | Dialog title |
| size | string | 'md' | Dialog size (sm, md, lg, xl) |
| loading | boolean | false | Loading state |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcDialogClose | {type} | Dialog closed |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| openDialog() | Promise<void> | Open dialog |
| closeDialog() | Promise<void> | Close dialog |
| isOpen() | Promise<boolean> | Check open state |

See [theming](../theming.md).

