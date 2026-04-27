# BcSetup — Global Configuration

Single entry point for all global config. Standalone — no framework dependency.

## Quick Start

```javascript
BcSetup.configure({
  baseUrl: '/api',
  auth: { type: 'bearer', token: () => localStorage.getItem('jwt') }
});
```

Zero config also works — all defaults are sensible.

## API

### `BcSetup.configure(partial)`

Merges partial config into current config. Can be called multiple times — each call merges, not replaces.

```typescript
BcSetup.configure({
  baseUrl: string;                    // Base URL for all API calls (default: '')
  headers: Record<string, string | (() => string)>;  // Extra headers for all requests
  auth: {
    type: 'bearer' | 'header' | 'cookie' | 'none';  // Auth strategy (default: 'none')
    token?: string | (() => string | null);           // For bearer type
    headerName?: string;                               // For header type
    headerValue?: string | (() => string | null);      // For header type
  };
  responseTransformer?: (response: any) => { data: any[]; total: number };  // Global response transformer
  validateOn: 'blur' | 'change' | 'submit' | 'manual';  // Default validation trigger (default: 'blur')
  validationMessages: Record<string, string>;              // Override default error messages
  size: 'sm' | 'md' | 'lg';           // Default field size (default: 'md')
  locale: string;                       // Locale for formatting (default: 'en')
  theme: 'light' | 'dark' | 'system' | string;  // Theme (default: 'light')
});
```

### `BcSetup.getConfig()`

Returns readonly snapshot of current config.

### `BcSetup.getHeaders()`

Returns resolved headers (auth + custom). Functions are called at resolve time.

### `BcSetup.getBaseUrl()`

Returns current base URL.

### `BcSetup.getValidationMessage(rule, ...params)`

Returns localized validation message. Supports `{0}`, `{1}` placeholders.

### `BcSetup.reactivity(rules)`

Register reactive field rules. See [reactivity.md](reactivity.md).

```javascript
BcSetup.reactivity({
  'type': (value, form) => {
    if (value === 'company') {
      form.setRequired('tax_id', true);
    }
  }
});
```

### `BcSetup.registerValidator(name, fn)`

Register a custom named validator. See [validation.md](validation.md).

```javascript
BcSetup.registerValidator('no-competitor', async (value) => {
  if (String(value).endsWith('@competitor.com')) return 'Competitor emails not allowed';
  return null;
});
```

### `BcSetup.reset()`

Reset all config to defaults. Useful for testing.

## Auto-Init from Meta Tags

BcSetup auto-reads meta tags on page load:

```html
<meta name="bc-base-url" content="/api">
<meta name="bc-auth-token" content="eyJhbG...">
<meta name="bc-theme" content="dark">
```

## Auth Examples

### Bearer Token (JWT)

```javascript
BcSetup.configure({
  auth: { type: 'bearer', token: () => localStorage.getItem('jwt') }
});
```

### Custom Header (API Key)

```javascript
BcSetup.configure({
  auth: {
    type: 'header',
    headerName: 'X-API-Key',
    headerValue: () => document.querySelector('meta[name=api-key]')?.content
  }
});
```

### Extra Headers

```javascript
BcSetup.configure({
  headers: {
    'X-Tenant': 'company-a',
    'Accept-Language': 'id'
  }
});
```

### Response Transformer

```javascript
BcSetup.configure({
  responseTransformer: (res) => ({
    data: res.results || res.data || [],
    total: res.total_count || res.total || 0
  })
});
```
