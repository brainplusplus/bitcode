# Media Viewer & Player Components — Design Document

**Date**: 2026-04-26
**Status**: Approved
**Scope**: 8 new Stencil Web Components + 2 component enhancements

---

## Overview

Add media viewer and player components to the BitCode component library. These are display-only components that render various media types inline. Additionally, enhance `bc-field-file` with preview/download capabilities and `bc-field-string` with social media embed widgets.

## New Components

All components live in `packages/components/src/components/widgets/` following the existing widget pattern (display-only, not form inputs).

### 1. `bc-viewer-pdf`

Renders PDF files inline using native browser PDF rendering.

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `src` | string | — | PDF file URL |
| `height` | string | `"600px"` | Viewer height |
| `toolbar` | boolean | `true` | Show browser PDF toolbar |
| `download` | boolean | `true` | Show download button |

**Implementation**: `<iframe src="URL#toolbar=1">` with `<object>` fallback for browsers that don't support iframe PDF.

### 2. `bc-viewer-image`

Image viewer with zoom and lightbox support.

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `src` | string | — | Image URL |
| `alt` | string | `""` | Alt text |
| `width` | string | `"100%"` | Container width |
| `height` | string | `"auto"` | Container height |
| `zoomable` | boolean | `true` | Enable click-to-zoom |
| `lightbox` | boolean | `true` | Enable fullscreen lightbox |
| `download` | boolean | `false` | Show download button |

**Implementation**: Native `<img>` with CSS transform zoom. Lightbox is a fullscreen overlay with backdrop blur.

### 3. `bc-viewer-document`

Office document viewer (doc, xls, ppt and their OpenXML variants).

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `src` | string | — | Document URL (must be publicly accessible) |
| `height` | string | `"600px"` | Viewer height |
| `provider` | string | `"microsoft"` | `"microsoft"` or `"google"` |
| `download` | boolean | `true` | Show download button |

**Implementation**: iframe to Microsoft Office Online viewer (`https://view.officeapps.live.com/op/embed.aspx?src=ENCODED_URL`) or Google Docs viewer (`https://docs.google.com/gview?url=ENCODED_URL&embedded=true`).

**Limitation**: File must be publicly accessible via URL for the iframe viewers to work. For private/local files, falls back to download link with file type icon.

### 4. `bc-viewer-youtube`

YouTube video embed supporting all URL formats and Shorts.

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `src` | string | — | YouTube URL or video ID |
| `width` | string | `"100%"` | Player width |
| `height` | string | `"auto"` | Player height (auto = 16:9 ratio) |
| `autoplay` | boolean | `false` | Auto-play video |
| `controls` | boolean | `true` | Show player controls |
| `start` | number | `0` | Start time in seconds |

**URL parsing** — extracts video ID from:
- `https://www.youtube.com/watch?v=VIDEO_ID`
- `https://youtu.be/VIDEO_ID`
- `https://www.youtube.com/shorts/VIDEO_ID`
- `https://www.youtube.com/embed/VIDEO_ID`
- `https://m.youtube.com/watch?v=VIDEO_ID`
- Raw video ID: `dQw4w9WgXcQ`

**Shorts detection**: If URL contains `/shorts/`, renders with 9:16 aspect ratio instead of 16:9.

**Implementation**: `<iframe src="https://www.youtube.com/embed/VIDEO_ID">` with responsive aspect-ratio container.

### 5. `bc-viewer-instagram`

Instagram post/reel embed.

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `src` | string | — | Instagram URL |
| `width` | string | `"400px"` | Embed width |
| `captioned` | boolean | `true` | Show caption |

**URL parsing** — extracts from:
- `https://www.instagram.com/p/POST_ID/`
- `https://www.instagram.com/reel/REEL_ID/`

**Implementation**: Instagram oEmbed blockquote + official `//www.instagram.com/embed.js` script injection.

### 6. `bc-viewer-tiktok`

TikTok video embed.

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `src` | string | — | TikTok URL |
| `width` | string | `"325px"` | Embed width |

**URL parsing** — extracts video ID from:
- `https://www.tiktok.com/@user/video/VIDEO_ID`
- `https://vm.tiktok.com/SHORT_ID/`

**Implementation**: TikTok oEmbed blockquote + official `//www.tiktok.com/embed.js` script injection.

### 7. `bc-viewer-video`

HTML5 video player with custom styled controls.

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `src` | string | — | Video URL |
| `type` | string | auto-detect | MIME type (`video/mp4`, `video/webm`, `video/ogg`) |
| `poster` | string | `""` | Poster image URL |
| `controls` | boolean | `true` | Show controls |
| `autoplay` | boolean | `false` | Auto-play |
| `loop` | boolean | `false` | Loop playback |
| `muted` | boolean | `false` | Start muted |
| `width` | string | `"100%"` | Player width |
| `height` | string | `"auto"` | Player height |
| `download` | boolean | `true` | Show download button |

**Supported formats**: mp4, webm, ogg

**Implementation**: Native `<video>` element with custom controls overlay (play/pause, seek bar, time display, volume, fullscreen, download).

### 8. `bc-viewer-audio`

HTML5 audio player with custom styled UI.

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `src` | string | — | Audio URL |
| `type` | string | auto-detect | MIME type |
| `controls` | boolean | `true` | Show controls |
| `autoplay` | boolean | `false` | Auto-play |
| `loop` | boolean | `false` | Loop playback |
| `download` | boolean | `true` | Show download button |

**Supported formats**: mp3, m4a, aac, ogg, webm

**Implementation**: Native `<audio>` element with custom player UI (play/pause, progress bar, time display, volume slider, download button).

---

## Existing Component Enhancements

### `bc-field-file` — Preview & Download

**New props:**
- `preview: boolean` (default `false`) — show inline viewer after upload
- `download: boolean` (default `true`) — show download button per file

**Viewer auto-detection** (by MIME type of uploaded file):

| MIME Pattern | Viewer |
|-------------|--------|
| `image/*` | `<bc-viewer-image>` |
| `application/pdf` | `<bc-viewer-pdf>` |
| `video/*` | `<bc-viewer-video>` |
| `audio/*` | `<bc-viewer-audio>` |
| `application/vnd.openxmlformats*`, `application/msword`, `application/vnd.ms-*` | `<bc-viewer-document>` |
| Other | Download link only |

### `bc-field-string` — Social Media Widgets

When used with `widget` prop, renders a text input + live preview:

| Widget Value | Behavior |
|-------------|----------|
| `youtube` | Text input for YouTube URL/ID + `<bc-viewer-youtube>` preview below |
| `instagram` | Text input for Instagram URL + `<bc-viewer-instagram>` preview below |
| `tiktok` | Text input for TikTok URL + `<bc-viewer-tiktok>` preview below |

Input is debounced (300ms) before updating the preview.

---

## Technical Decisions

### Zero External Dependencies

All components use native browser APIs:
- PDF: browser `<iframe>`/`<object>`
- Image: `<img>` + CSS transforms
- Video/Audio: `<video>`/`<audio>` elements
- YouTube: iframe embed API
- Instagram/TikTok: official oEmbed scripts

### Shadow DOM

All components use `shadow: true` (consistent with existing components).

### CSS Variables

All components use the existing CSS variable system (`--bc-*` variables) from `field-base.css` and the global design system.

### i18n

All user-facing strings use the existing i18n system. Translations provided for all 11 languages: en, id, ar, de, es, fr, ja, ko, pt-BR, ru, zh-CN.

### Component Compiler Integration

No changes needed to `component_compiler.go` — these are widget/viewer components used programmatically, not mapped from field types. The file upload preview integration happens within the Stencil component itself.

---

## File Structure

```
packages/components/src/components/widgets/
├── bc-viewer-pdf/
│   ├── bc-viewer-pdf.tsx
│   └── bc-viewer-pdf.css
├── bc-viewer-image/
│   ├── bc-viewer-image.tsx
│   └── bc-viewer-image.css
├── bc-viewer-document/
│   ├── bc-viewer-document.tsx
│   └── bc-viewer-document.css
├── bc-viewer-youtube/
│   ├── bc-viewer-youtube.tsx
│   └── bc-viewer-youtube.css
├── bc-viewer-instagram/
│   ├── bc-viewer-instagram.tsx
│   └── bc-viewer-instagram.css
├── bc-viewer-tiktok/
│   ├── bc-viewer-tiktok.tsx
│   └── bc-viewer-tiktok.css
├── bc-viewer-video/
│   ├── bc-viewer-video.tsx
│   └── bc-viewer-video.css
└── bc-viewer-audio/
    ├── bc-viewer-audio.tsx
    └── bc-viewer-audio.css
```
