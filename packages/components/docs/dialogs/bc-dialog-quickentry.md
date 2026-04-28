# bc-dialog-quickentry

> Quick create dialog with auto-generated form fields

## Quick Start

```html
<bc-dialog-quickentry></bc-dialog-quickentry>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| open | boolean | false | Open state |
| dialog-title | string | '' | Title |
| model | string | '' | Model name for API create |
| fields | string (JSON) | '[]' | Field names to show |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcDialogClose | {type, data?} | Closed (with created data if saved) |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| openDialog() | Promise<void> | Open |
| closeDialog() | Promise<void> | Close |

See [theming](../theming.md).

