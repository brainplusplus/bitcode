# bc-view-list

> List view with pagination and row selection

## Quick Start

```html
<bc-view-list></bc-view-list>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| model | string | '' | Model name |
| view-title | string | '' | List title |
| fields | string (JSON) | '[]' | Field definitions |
| config | string (JSON) | '{}' | View config |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcRowSelect | {ids} | Rows selected |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| refresh() | Promise<void> | Reload list |

See [theming](../theming.md).

