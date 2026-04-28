# bc-viewer-youtube

> YouTube video embed

## Quick Start

```html
<bc-viewer-youtube></bc-viewer-youtube>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| src | string | '' | YouTube URL or video ID |
| width | string | '100%' | Width |
| height | string | 'auto' | Height |
| autoplay | boolean | false | Auto-play |
| controls | boolean | true | Show controls |
| start | number | 0 | Start time (seconds) |
| loading | boolean | false | Loading state |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| refresh() | Promise<void> | Reload |

See [theming](../theming.md).

