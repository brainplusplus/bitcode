# bc-view-map

> Map view with markers (Leaflet)

## Quick Start

```html
<bc-view-map></bc-view-map>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| model | string | '' | Model name |
| view-title | string | '' | Title |
| fields | string (JSON) | '[]' | Fields |
| config | string (JSON) | '{}' | Config |
| geo-field | string | 'location' | Geo field name |
| name-field | string | 'name' | Name field |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| refresh() | Promise<void> | Reload |

See [theming](../theming.md).

