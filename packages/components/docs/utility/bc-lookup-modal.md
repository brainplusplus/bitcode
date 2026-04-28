# bc-lookup-modal

> Record lookup modal with search and selection

## Quick Start

```html
<bc-lookup-modal></bc-lookup-modal>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| open | boolean | false | Open state |
| model | string | '' | Model name |
| display-field | string | 'name' | Display field |
| columns | string (JSON) | '[]' | Table columns |
| multiple | boolean | false | Multi-select |
| api-url | string | '' | Custom API URL |
| modal-title | string | '' | Modal title |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcLookupSelect | {records} | Records selected |
| lcLookupClose | {} | Modal closed |

See [theming](../theming.md).

