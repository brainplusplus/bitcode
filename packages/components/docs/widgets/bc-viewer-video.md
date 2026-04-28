# bc-viewer-video

> HTML5 video player with custom controls

## Quick Start

```html
<bc-viewer-video></bc-viewer-video>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| src | string | '' | Video URL |
| type | string | '' | MIME type |
| poster | string | '' | Poster image |
| controls | boolean | true | Show controls |
| autoplay | boolean | false | Auto-play |
| loop | boolean | false | Loop |
| muted | boolean | false | Muted |
| width | string | '100%' | Width |
| height | string | 'auto' | Height |
| download | boolean | true | Show download |
| loading | boolean | false | Loading state |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| refresh() | Promise<void> | Reload |

See [theming](../theming.md).

