# Getting Started

## Installation

### CDN (no build step)

```html
<script type="module" src="https://unpkg.com/@bitcode/components/dist/bc-components/bc-components.esm.js"></script>
```

### NPM

```bash
npm install @bitcode/components
```

```javascript
import { defineCustomElements } from '@bitcode/components/loader';
defineCustomElements();
```

## Basic Usage

```html
<bc-field-string name="email" label="Email" required placeholder="you@example.com" />
<bc-field-integer name="age" label="Age" min="0" max="150" />
<bc-field-select name="country" label="Country" options='[{"label":"Indonesia","value":"ID"},{"label":"Japan","value":"JP"}]' />
```

Components work standalone — no framework, no backend, no config needed.

## With API Data

```javascript
BcSetup.configure({
  baseUrl: '/api',
  auth: { type: 'bearer', token: () => localStorage.getItem('jwt') }
});
```

```html
<bc-field-select name="city" label="City" data-source="/api/cities" searchable />
<bc-datatable columns='[{"field":"name","header":"Name"},{"field":"email","header":"Email"}]' data-source="/api/users" server-side />
```

## Dark Mode

```html
<body data-bc-theme="dark">
```

Or auto-detect system preference:

```javascript
BcSetup.configure({ theme: 'system' });
```

## Framework Integration

### React

```jsx
import { defineCustomElements } from '@bitcode/components/loader';
defineCustomElements();

function App() {
  return <bc-field-string name="email" label="Email" required />;
}
```

### Vue

```javascript
import { defineCustomElements } from '@bitcode/components/loader';
defineCustomElements();

// vue.config.js or vite.config.js
app.config.compilerOptions.isCustomElement = (tag) => tag.startsWith('bc-');
```

### Angular

```typescript
// app.module.ts
import { CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';

@NgModule({
  schemas: [CUSTOM_ELEMENTS_SCHEMA]
})
```

## Next Steps

- [BcSetup](bc-setup.md) — global configuration
- [Theming](theming.md) — light, dark, custom themes
- [Data Fetching](data-fetching.md) — 4-level data strategy
- [Validation](validation.md) — 3-level validation
- [Reactivity](reactivity.md) — dependent fields
