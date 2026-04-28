# bc-viewer-image

> Image viewer with zoom and lightbox

## Quick Start

```html
<bc-viewer-image></bc-viewer-image>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| src | string | '' | Image URL |
| alt | string | '' | Alt text |
| width | string | '100%' | Width |
| height | string | 'auto' | Height |
| zoomable | boolean | true | Enable zoom |
| lightbox | boolean | true | Enable lightbox |
| download | boolean | false | Show download |
| loading | boolean | false | Loading state |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| refresh() | Promise<void> | Reload image |

See [theming](../theming.md).

