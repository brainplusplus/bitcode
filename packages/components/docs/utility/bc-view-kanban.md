# bc-view-kanban

> Kanban board view

## Quick Start

```html
<bc-view-kanban></bc-view-kanban>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| model | string | '' | Model name |
| view-title | string | '' | Title |
| fields | string (JSON) | '[]' | Field definitions |
| config | string (JSON) | '{}' | View config |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcKanbanMove | {id, from, to} | Card moved |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| refresh() | Promise<void> | Reload |

See [theming](../theming.md).

