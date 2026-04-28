# bc-child-table

> Inline editable child table (one2many)

## Quick Start

```html
<bc-child-table></bc-child-table>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| field | string | '' | Field name |
| columns | string (JSON) | '[]' | Column definitions |
| data | string (JSON) | '[]' | Table data |
| summary | string (JSON) | '{}' | Summary config |
| readonly | boolean | false | Read-only mode |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcFieldChange | {name, value, oldValue} | Data changed |

See [theming](../theming.md).

