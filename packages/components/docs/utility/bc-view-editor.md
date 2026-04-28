# bc-view-editor

> Visual form layout editor (drag-and-drop)

## Quick Start

```html
<bc-view-editor></bc-view-editor>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| view-json | string | '{}' | Layout JSON |
| model-fields | string | '[]' | Available fields |
| readonly | boolean | false | Read-only mode |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| viewChanged | {json} | Layout changed |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| refresh() | Promise<void> | Reload |

See [theming](../theming.md).

