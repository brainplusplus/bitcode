# Phase 1: Core Infrastructure — Design + Implementation Plan

**Date:** 2026-07-27
**Depends on:** Nothing (foundation phase)
**Blocks:** All other phases

---

## SCOPE

Create the shared infrastructure that ALL component upgrades depend on:

1. **`BcSetup`** — Global configuration (auth, headers, base URL, response transformer, **theme**)
2. **`data-fetcher.ts`** — 4-level data fetching utility (standalone, no BitCode dependency)
3. **`field-utils.ts`** — Shared field logic (validation, dirty/touched, ARIA, render helpers)
4. **`types.ts`** — New event interfaces and shared types
5. **`validation-engine.ts`** — 3-level validation (built-in, custom JS, server-side)
6. **Theming system** — light/dark/system-detect/custom themes via CSS custom properties

---

## 1. BcSetup — Global Configuration

### File: `packages/components/src/core/bc-setup.ts`

Single entry point for all global config. Standalone — no BitCode dependency.

```typescript
export interface BcConfig {
  // --- API ---
  baseUrl: string;
  headers: Record<string, string | (() => string)>;

  // --- Auth ---
  auth: {
    type: 'bearer' | 'header' | 'cookie' | 'none';
    token?: string | (() => string | null);     // For bearer
    headerName?: string;                         // For header type
    headerValue?: string | (() => string | null); // For header type
  };

  // --- Response ---
  responseTransformer?: (response: any) => { data: any[]; total: number };

  // --- Validation ---
  validateOn: 'blur' | 'change' | 'submit' | 'manual';  // Global default
  validationMessages: Record<string, string>;              // i18n override

  // --- UI ---
  size: 'sm' | 'md' | 'lg';           // Global default field size
  locale: string;                       // For formatting

  // --- Theme ---
  theme: 'light' | 'dark' | 'system' | string;  // Built-in or custom theme name
}

class BcSetupClass {
  private config: BcConfig = { /* sensible defaults */ };
  private reactivityRules: Map<string, Function> = new Map();
  private customValidators: Map<string, Function> = new Map();

  configure(partial: Partial<BcConfig>): void { ... }
  getConfig(): Readonly<BcConfig> { ... }

  // Resolve headers for any fetch call
  getHeaders(): Record<string, string> { ... }

  // Reactivity rules (Phase 6, but API defined here)
  reactivity(rules: Record<string, (value: any, form: FormProxy) => void>): void { ... }
  getReactivityRule(fieldName: string): Function | undefined { ... }

  // Custom validators (Phase 6, but API defined here)
  registerValidator(name: string, fn: (value: any) => string | null | Promise<string | null>): void { ... }
  getValidator(name: string): Function | undefined { ... }

  // Reset (for testing)
  reset(): void { ... }
}

export const BcSetup: BcSetupClass;
```

### Auto-init from meta tags:
```typescript
// On module load, auto-read meta tags
if (typeof document !== 'undefined') {
  const baseUrl = document.querySelector('meta[name="bc-base-url"]')?.getAttribute('content');
  const token = document.querySelector('meta[name="bc-auth-token"]')?.getAttribute('content');
  if (baseUrl) BcSetup.configure({ baseUrl });
  if (token) BcSetup.configure({ auth: { type: 'bearer', token } });
}
```

### Usage examples:

```javascript
// Minimal (most common)
BcSetup.configure({
  baseUrl: '/api',
  auth: { type: 'bearer', token: () => localStorage.getItem('jwt') }
});

// Full custom
BcSetup.configure({
  baseUrl: 'https://api.myapp.com/v2',
  headers: {
    'X-Tenant': 'company-a',
    'Accept-Language': 'id'
  },
  auth: {
    type: 'header',
    headerName: 'X-API-Key',
    headerValue: () => document.querySelector('meta[name=api-key]')?.content
  },
  responseTransformer: (res) => ({
    data: res.results || res.data || res.items || [],
    total: res.total_count || res.total || res.count || 0
  }),
  validateOn: 'blur',
  size: 'md',
  locale: 'id-ID'
});

// Zero config also works — all defaults are sensible
```

---

## 2. data-fetcher.ts — 4-Level Data Fetching

### File: `packages/components/src/core/data-fetcher.ts`

```typescript
// === TYPES ===

export interface FetchParams {
  page?: number;
  pageSize?: number;
  sort?: Array<{ field: string; direction: 'asc' | 'desc' }>;
  filters?: Record<string, unknown>;
  search?: string;
  dependValues?: Record<string, unknown>;
}

export interface FetchResult {
  data: any[];
  total: number;
}

export type DataFetcher = (params: FetchParams) => Promise<FetchResult>;
export type OptionsFetcher = (query: string, params: FetchParams) => Promise<any[]>;

// === MAIN FUNCTIONS ===

/**
 * Fetch data using 4-level priority:
 * 1. Custom fetcher function (Level 4)
 * 2. Event intercept + dataSource URL (Level 3)
 * 3. dataSource URL with native fetch (Level 2)
 * 4. Local data from prop (Level 1)
 * 5. BitCode api-client fallback (if available)
 */
export async function fetchData(opts: {
  fetcher?: DataFetcher;
  element?: HTMLElement;
  dataSource?: string;
  localData?: string | any[];
  model?: string;
  fetchHeaders?: string | Record<string, string>;
  params?: FetchParams;
}): Promise<FetchResult>

/**
 * Fetch options for select-family components.
 * Same 4-level priority.
 */
export async function fetchOptions(opts: {
  fetcher?: OptionsFetcher;
  element?: HTMLElement;
  dataSource?: string;
  localOptions?: string | any[];
  model?: string;
  query?: string;
  fetchHeaders?: string | Record<string, string>;
  params?: FetchParams;
}): Promise<any[]>

/**
 * Auto-detect response format and normalize.
 * Handles: { data: [...] }, [...], { results: [...] }, { items: [...] },
 *          { records: [...] }, { rows: [...] }
 * Also checks BcSetup.responseTransformer if set.
 */
export function normalizeResponse(response: any): FetchResult

/**
 * Resolve URL template: "/api/cities?province={province}" + values
 */
export function resolveUrl(template: string, values: Record<string, unknown>): string

/**
 * Build headers: merge BcSetup headers + auth + custom headers
 */
export function buildHeaders(customHeaders?: string | Record<string, string>): Record<string, string>
```

### Internal flow for fetchData:

```
fetchData(opts) {
  // Level 4: Custom fetcher
  if (opts.fetcher) return opts.fetcher(opts.params);

  // Level 3 + 2: dataSource URL
  if (opts.dataSource) {
    const url = resolveUrl(opts.dataSource, opts.params?.dependValues || {});
    const headers = buildHeaders(opts.fetchHeaders);

    // Level 3: Fire lcBeforeFetch event — let consumer modify
    if (opts.element) {
      const beforeEvent = new CustomEvent('lcBeforeFetch', {
        detail: { url, headers, params: opts.params },
        bubbles: true, cancelable: true
      });
      opts.element.dispatchEvent(beforeEvent);
      // Consumer may have modified detail.url, detail.headers
    }

    // Fetch
    const response = await fetch(url, { headers });
    const json = await response.json();

    // Level 3: Fire lcAfterFetch event — let consumer transform
    if (opts.element) {
      const afterEvent = new CustomEvent('lcAfterFetch', {
        detail: { response: json, data: null, total: 0 },
        bubbles: true
      });
      opts.element.dispatchEvent(afterEvent);
      if (afterEvent.detail.data) {
        return { data: afterEvent.detail.data, total: afterEvent.detail.total };
      }
    }

    // Auto-normalize
    return normalizeResponse(json);
  }

  // Level 1: Local data
  if (opts.localData) {
    const data = typeof opts.localData === 'string' ? JSON.parse(opts.localData) : opts.localData;
    return { data, total: data.length };
  }

  // BitCode fallback
  if (opts.model) {
    try {
      const { getApiClient } = await import('./api-client');
      const api = getApiClient();
      const result = await api.list(opts.model, opts.params as any);
      return { data: result.data, total: result.total };
    } catch {
      // api-client not available — standalone mode
    }
  }

  return { data: [], total: 0 };
}
```

---

## 3. field-utils.ts — Shared Field Utilities

### File: `packages/components/src/core/field-utils.ts`

```typescript
// === VALIDATION ===

export interface ValidationRule {
  rule: string | ((value: any) => boolean | Promise<boolean>);
  message: string;
}

export interface ValidationResult {
  valid: boolean;
  errors: string[];
}

/**
 * Run built-in validation rules from props.
 * Sync — runs instantly.
 */
export function validateBuiltIn(value: any, opts: {
  required?: boolean;
  minLength?: number;
  maxLength?: number;
  min?: number;
  max?: number;
  pattern?: string;
  patternMessage?: string;
  type?: string;  // 'email', 'url', 'phone' — auto-add pattern
}): ValidationResult

/**
 * Run custom validators array.
 * Can be async.
 */
export async function validateCustom(
  value: any,
  validators: ValidationRule[]
): Promise<ValidationResult>

/**
 * Run server-side validator.
 * Debounced internally.
 */
export async function validateServer(
  value: any,
  validator: string | ((value: any) => Promise<string | null>),
  headers?: Record<string, string>
): Promise<ValidationResult>

/**
 * Full validation pipeline: built-in → custom → server.
 * Stops at first failure level.
 */
export async function validateField(value: any, opts: {
  builtIn: Parameters<typeof validateBuiltIn>[1];
  custom?: ValidationRule[];
  server?: string | ((value: any) => Promise<string | null>);
}): Promise<ValidationResult>


// === DIRTY/TOUCHED TRACKING ===

export interface FieldState {
  dirty: boolean;
  touched: boolean;
  pristine: boolean;
  initialValue: any;
}

export function createFieldState(initialValue: any): FieldState
export function markDirty(state: FieldState, currentValue: any): FieldState
export function markTouched(state: FieldState): FieldState
export function resetState(state: FieldState, newInitialValue?: any): FieldState


// === ARIA ===

export function getAriaAttrs(props: {
  name: string;
  required?: boolean;
  disabled?: boolean;
  readonly?: boolean;
  validationStatus?: string;
  validationMessage?: string;
  hint?: string;
}): Record<string, string>


// === RENDER HELPERS (return JSX-compatible objects) ===

export function renderHint(hint: string): any          // <div class="bc-field-hint">...</div>
export function renderError(message: string): any      // <div class="bc-field-error">...</div>
export function renderCounter(current: number, max: number): any
export function renderClearButton(onClick: () => void): any
export function renderPrefix(prefix: string): any
export function renderSuffix(suffix: string): any
export function renderTooltip(tooltip: string): any
export function renderLoading(): any

/**
 * Get CSS classes for field wrapper based on state.
 */
export function getFieldClasses(opts: {
  size?: string;
  validationStatus?: string;
  disabled?: boolean;
  readonly?: boolean;
  loading?: boolean;
  dirty?: boolean;
  touched?: boolean;
}): Record<string, boolean>
```

---

## 4. types.ts — New Event Interfaces

### Add to existing `packages/components/src/core/types.ts`:

```typescript
// --- Field Events ---
export interface FieldFocusEvent {
  name: string;
  value: unknown;
}

export interface FieldBlurEvent {
  name: string;
  value: unknown;
  dirty: boolean;
  touched: boolean;
}

export interface FieldClearEvent {
  name: string;
  oldValue: unknown;
}

export interface FieldValidationEvent {
  name: string;
  value: unknown;
  errors: string[];
}

export interface FieldValidEvent {
  name: string;
  value: unknown;
}

// --- Data Fetch Events ---
export interface BeforeFetchEvent {
  url: string;
  headers: Record<string, string>;
  params: any;
}

export interface AfterFetchEvent {
  response: any;
  data: any[] | null;
  total: number;
}

// --- Chart Events ---
export interface ChartClickEvent {
  name: string;
  value: unknown;
  dataIndex: number;
}

// --- DataTable Events ---
export interface RowEditEvent {
  row: Record<string, unknown>;
  field: string;
  value: unknown;
  oldValue: unknown;
}

export interface CellClickEvent {
  row: Record<string, unknown>;
  column: string;
  value: unknown;
}

export interface ColumnResizeEvent {
  column: string;
  width: number;
}

export interface PageChangeEvent {
  page: number;
  pageSize: number;
}

// --- Select Events ---
export interface OptionsLoadEvent {
  options: unknown[];
  total: number;
}

export interface OptionCreateEvent {
  value: string;
}
```

---

## 5. validation-engine.ts — Validation Pipeline

### File: `packages/components/src/core/validation-engine.ts`

This is the orchestrator that field-utils.ts calls. Separated for clarity.

```typescript
/**
 * Built-in validator registry.
 * Pre-loaded with: required, email, url, phone, minLength, maxLength, min, max, pattern
 * Extensible via BcSetup.registerValidator()
 */
export const builtInValidators: Map<string, (value: any, param?: any) => string | null>;

/**
 * Validate a single value through the full pipeline.
 *
 * Execution order:
 * 1. Built-in rules (sync) — required, minLength, pattern, etc.
 * 2. Custom validators (can be async) — from validators JS property
 * 3. Server-side validator (async, debounced) — from serverValidator JS property
 *
 * Stops at first level that has errors (doesn't run server if built-in fails).
 */
export async function runValidationPipeline(opts: {
  value: any;
  fieldName: string;

  // Level 1: Built-in (from props)
  required?: boolean;
  minLength?: number;
  maxLength?: number;
  min?: number;
  max?: number;
  pattern?: string;
  patternMessage?: string;

  // Level 2: Custom (from JS property)
  validators?: Array<{
    rule: string | ((value: any) => boolean | Promise<boolean>);
    message: string;
  }>;
  customValidator?: (value: any) => string | null | Promise<string | null>;

  // Level 3: Server-side (from JS property)
  serverValidator?: string | ((value: any) => Promise<string | null>);
  serverValidatorHeaders?: Record<string, string>;
}): Promise<{ valid: boolean; errors: string[] }>
```

---

## 6. THEMING SYSTEM

### Problem
- No dark mode
- No system preference detection
- No easy way to switch themes
- Shadow DOM inconsistent (21 true, 13 false) — theming behavior unpredictable
- No size tokens for sm/md/lg
- No CSS variables for new enterprise features (validation states, dropdown, tooltip, etc)

### Decision: Shadow DOM → ALL `shadow: false`

All 103 components will use `shadow: false`. Reasons:
- Theming becomes trivial — standard CSS works
- All classes prefixed `bc-*` — conflict risk minimal
- Consistent behavior across all components
- Matches Odoo, Frappe, Shoelace pattern
- `::part()` not needed — direct CSS access

This is a **Phase 2 execution task** (change shadow flag per component), but the **decision** is made here.

### Architecture

```
global/
├── global.css              # Base variables (light theme = default)
├── themes/
│   ├── dark.css            # Dark theme overrides
│   └── README.md           # How to create custom themes
```

No extra files to load for light theme — it's the default in `global.css`.
Dark theme is a small override file (~100 vars).
Custom themes follow the same pattern.

### Theme Switching — 3 Mechanisms

#### Mechanism 1: `data-bc-theme` attribute (recommended)
```html
<!-- Light (default, no attribute needed) -->
<body>

<!-- Dark -->
<body data-bc-theme="dark">

<!-- Scoped — only this section is dark -->
<div data-bc-theme="dark">
  <bc-field-string name="x" label="Dark field" />
</div>

<!-- Custom theme -->
<body data-bc-theme="corporate">
```

#### Mechanism 2: `BcSetup.configure()` (programmatic)
```javascript
BcSetup.configure({ theme: 'dark' });
// Internally: document.documentElement.setAttribute('data-bc-theme', 'dark')

BcSetup.configure({ theme: 'system' });
// Internally: listen to prefers-color-scheme, auto-switch

BcSetup.configure({ theme: 'light' });
// Internally: remove attribute (light is default)
```

#### Mechanism 3: CSS `prefers-color-scheme` (auto, zero JS)
```css
/* In global.css — auto dark mode without any JS */
@media (prefers-color-scheme: dark) {
  :root:not([data-bc-theme]) {
    /* dark overrides here */
    /* Only applies when NO explicit theme is set */
    /* If developer sets data-bc-theme="light", this is ignored */
  }
}
```

#### Priority:
```
1. data-bc-theme attribute on nearest ancestor  → highest (scoped)
2. data-bc-theme on <html>/<body>               → page-level
3. BcSetup.configure({ theme })                 → sets attribute on <html>
4. @media prefers-color-scheme                   → auto fallback
5. :root defaults                                → light theme
```

### System Detection Implementation

```typescript
// In bc-setup.ts
private applyTheme(theme: string): void {
  if (theme === 'system') {
    // Detect and listen
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const apply = (e: MediaQueryListEvent | MediaQueryList) => {
      document.documentElement.setAttribute('data-bc-theme', e.matches ? 'dark' : 'light');
    };
    apply(mq);
    mq.addEventListener('change', apply);
    this._systemThemeCleanup = () => mq.removeEventListener('change', apply);
  } else if (theme === 'light') {
    document.documentElement.removeAttribute('data-bc-theme');
  } else {
    document.documentElement.setAttribute('data-bc-theme', theme);
  }
}
```

### Dark Theme Variables

File: `global/themes/dark.css`

```css
[data-bc-theme="dark"] {
  /* Colors — Primary (lighter for dark bg) */
  --bc-primary: #818cf8;
  --bc-primary-hover: #a5b4fc;
  --bc-primary-light: rgba(129, 140, 248, 0.15);
  --bc-primary-text: #ffffff;

  /* Colors — Semantic */
  --bc-success: #34d399;
  --bc-success-light: rgba(52, 211, 153, 0.15);
  --bc-warning: #fbbf24;
  --bc-warning-light: rgba(251, 191, 36, 0.15);
  --bc-danger: #f87171;
  --bc-danger-light: rgba(248, 113, 113, 0.15);
  --bc-info: #60a5fa;
  --bc-info-light: rgba(96, 165, 250, 0.15);
  --bc-muted: #9ca3af;

  /* Background */
  --bc-bg: #0f172a;
  --bc-bg-secondary: #1e293b;
  --bc-bg-tertiary: #334155;
  --bc-bg-hover: #1e293b;

  /* Borders */
  --bc-border-color: #334155;
  --bc-border-color-focus: #818cf8;

  /* Text */
  --bc-text: #f1f5f9;
  --bc-text-secondary: #94a3b8;
  --bc-text-placeholder: #64748b;
  --bc-text-disabled: #475569;

  /* Shadows (subtle on dark) */
  --bc-shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.3);
  --bc-shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.4);
  --bc-shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.5);

  /* Input */
  --bc-input-bg: #1e293b;
  --bc-input-focus-ring: 0 0 0 3px rgba(129, 140, 248, 0.25);
}
```

### Size Tokens

Add to `global.css`:

```css
:root {
  /* Size: SM */
  --bc-input-height-sm: 1.875rem;
  --bc-input-padding-x-sm: 0.5rem;
  --bc-input-padding-y-sm: 0.25rem;
  --bc-font-size-input-sm: 0.8125rem;
  --bc-label-size-sm: 0.75rem;

  /* Size: MD (current defaults, keep as-is) */
  --bc-input-height-md: 2.5rem;
  --bc-input-padding-x-md: 0.75rem;
  --bc-input-padding-y-md: 0.5rem;
  --bc-font-size-input-md: 0.875rem;
  --bc-label-size-md: 0.875rem;

  /* Size: LG */
  --bc-input-height-lg: 3rem;
  --bc-input-padding-x-lg: 1rem;
  --bc-input-padding-y-lg: 0.625rem;
  --bc-font-size-input-lg: 1rem;
  --bc-label-size-lg: 1rem;
}
```

### Enterprise Feature Variables

Add to `global.css`:

```css
:root {
  /* Validation states */
  --bc-field-valid-color: var(--bc-success);
  --bc-field-valid-border: var(--bc-success);
  --bc-field-invalid-color: var(--bc-danger);
  --bc-field-invalid-border: var(--bc-danger);
  --bc-field-validating-color: var(--bc-info);
  --bc-field-warning-color: var(--bc-warning);

  /* Hint text */
  --bc-field-hint-color: var(--bc-text-secondary);
  --bc-field-hint-size: var(--bc-font-size-xs);

  /* Counter */
  --bc-field-counter-color: var(--bc-text-secondary);

  /* Prefix/Suffix addon */
  --bc-field-addon-bg: var(--bc-bg-tertiary);
  --bc-field-addon-color: var(--bc-text-secondary);
  --bc-field-addon-border: var(--bc-border-color);

  /* Clear button */
  --bc-field-clear-color: var(--bc-text-secondary);
  --bc-field-clear-hover: var(--bc-text);

  /* Loading spinner */
  --bc-field-loading-color: var(--bc-primary);

  /* Tooltip */
  --bc-tooltip-bg: var(--bc-text);
  --bc-tooltip-color: var(--bc-bg);
  --bc-tooltip-radius: var(--bc-radius-md);
  --bc-tooltip-padding: 0.375rem 0.625rem;
  --bc-tooltip-size: var(--bc-font-size-xs);

  /* Dropdown (select, tags, autocomplete) */
  --bc-dropdown-bg: var(--bc-bg);
  --bc-dropdown-border: var(--bc-border-color);
  --bc-dropdown-shadow: var(--bc-shadow-lg);
  --bc-dropdown-item-hover: var(--bc-bg-hover);
  --bc-dropdown-item-selected: var(--bc-primary-light);
  --bc-dropdown-item-padding: 0.5rem 0.75rem;
  --bc-dropdown-max-height: 15rem;
  --bc-dropdown-radius: var(--bc-radius-lg);
  --bc-dropdown-z: var(--bc-z-dropdown);

  /* Tag/Badge */
  --bc-tag-bg: var(--bc-bg-tertiary);
  --bc-tag-color: var(--bc-text);
  --bc-tag-radius: var(--bc-radius-full);
  --bc-tag-padding: 0.125rem 0.5rem;
  --bc-tag-size: var(--bc-font-size-xs);

  /* Table */
  --bc-table-header-bg: var(--bc-bg-secondary);
  --bc-table-header-color: var(--bc-text-secondary);
  --bc-table-row-hover: var(--bc-bg-hover);
  --bc-table-row-selected: var(--bc-primary-light);
  --bc-table-border: var(--bc-border-color);
  --bc-table-stripe-bg: var(--bc-bg-secondary);

  /* Chart */
  --bc-chart-bg: var(--bc-bg);
  --bc-chart-text: var(--bc-text);
  --bc-chart-grid: var(--bc-border-color);
  --bc-chart-colors: #4f46e5, #10b981, #f59e0b, #ef4444, #8b5cf6, #06b6d4, #f97316, #ec4899;
}
```

### Custom Theme — How Easy?

Developer creates a custom theme in **one CSS block**:

```css
/* corporate-theme.css — override only what you need */
[data-bc-theme="corporate"] {
  --bc-primary: #003366;
  --bc-primary-hover: #004488;
  --bc-font-family: 'Segoe UI', Tahoma, sans-serif;
  --bc-radius-sm: 0;
  --bc-radius-md: 0;
  --bc-radius-lg: 2px;
  /* Everything else inherits from light defaults */
}
```

Apply:
```html
<link rel="stylesheet" href="corporate-theme.css">
<body data-bc-theme="corporate">
```

Or scoped:
```html
<div data-bc-theme="corporate">
  <bc-field-string name="x" />  <!-- corporate styled -->
</div>
<bc-field-string name="y" />    <!-- default light styled -->
```

That's it. No build step. No Tailwind. No JS config. Just CSS.

---

## IMPLEMENTATION ORDER (UPDATED)

| Step | File | What | Est. Lines |
|------|------|------|-----------|
| 1 | `core/types.ts` | Add new event interfaces | ~80 |
| 2 | `global/global.css` | Add size tokens, enterprise vars, system dark media query | ~120 new |
| 3 | `global/themes/dark.css` | Dark theme overrides | ~80 |
| 4 | `core/bc-setup.ts` | Global config singleton + theme switching + system detect | ~200 |
| 5 | `core/data-fetcher.ts` | 4-level data fetching | ~200 |
| 6 | `core/validation-engine.ts` | 3-level validation pipeline | ~150 |
| 7 | `core/field-utils.ts` | Shared field utilities + size-aware render helpers | ~280 |
| 8 | Build & test | `npm run build` | — |

**Total: ~1110 lines of new/modified core code.**

After Phase 1, all other phases can start. Phase 2-5 can even run in parallel since they all depend only on Phase 1.

---

## WHAT THIS PHASE DOES NOT INCLUDE

- Actual component changes (Phase 2-5)
- Shadow DOM standardization (Phase 2 — change `shadow: true` → `false` per component)
- `BcSetup.reactivity()` implementation (Phase 6 — API defined here, impl later)
- FormProxy / cross-field reactivity (Phase 6)
- Any changes to engine Go (never)

---

**Generated: 2026-07-27**
