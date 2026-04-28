# bc-filter-bar

> Quick filter bar with presets

## Quick Start

```html
<bc-filter-bar></bc-filter-bar>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| value | string | '' | Current filter |
| presets | string (JSON) | '[]' | Filter presets |
| placeholder | string | 'Search...' | Placeholder |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcFilterChange | {filters} | Filter changed |
| lcSearch | {query} | Search |

See [theming](../theming.md).

