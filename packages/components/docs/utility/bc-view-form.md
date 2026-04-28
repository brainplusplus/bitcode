# bc-view-form

> Form view with CRUD operations

## Quick Start

```html
<bc-view-form></bc-view-form>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| model | string | '' | Model name |
| view-title | string | '' | Form title |
| record-id | string | '' | Record ID (empty=new) |
| fields | string (JSON) | '[]' | Field definitions |
| config | string (JSON) | '{}' | View config |
| permissions | string (JSON) | '{}' | CRUD permissions |
| module-name | string | '' | Module name |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcFormSubmit | {model, data, id?} | Form submitted |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| refresh() | Promise<void> | Reload form data |

See [theming](../theming.md).

