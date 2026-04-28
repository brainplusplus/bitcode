# bc-section

> Collapsible section with title and description

## Quick Start

```html
<bc-section></bc-section>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| section-title | string | '' | Section heading |
| description | string | '' | Section description |
| collapsible | boolean | false | Enable collapse |
| collapsed | boolean | false | Initial collapsed state |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| toggle() | Promise<void> | Toggle collapse |
| expand() | Promise<void> | Expand section |
| collapse() | Promise<void> | Collapse section |

See [theming](../theming.md).

