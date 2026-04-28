# bc-filter-builder

> Advanced filter builder with conditions and groups

## Quick Start

```html
<bc-filter-builder></bc-filter-builder>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| fields | string (JSON) | '[]' | Available fields |
| operators | string (JSON) | '[...]' | Available operators |
| value | string | '' | Current filter JSON |
| show-json-toggle | boolean | false | Show JSON view |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcFilterChange | {filter} | Filter changed |

See [theming](../theming.md).

