# bc-viewer-document

> Document viewer (Office files via Microsoft/Google)

## Quick Start

```html
<bc-viewer-document></bc-viewer-document>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| src | string | '' | Document URL |
| height | string | '600px' | Viewer height |
| provider | string | 'microsoft' | 'microsoft' or 'google' |
| download | boolean | true | Show download |
| loading | boolean | false | Loading state |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| refresh() | Promise<void> | Reload |

See [theming](../theming.md).

