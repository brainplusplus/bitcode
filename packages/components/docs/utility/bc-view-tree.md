# bc-view-tree

> Tree/hierarchy view

## Quick Start

```html
<bc-view-tree></bc-view-tree>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| model | string | '' | Model name |
| view-title | string | '' | Title |
| fields | string (JSON) | '[]' | Fields |
| config | string (JSON) | '{}' | Config |
| parent-field | string | 'parent_id' | Parent field name |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| refresh() | Promise<void> | Reload |

See [theming](../theming.md).

