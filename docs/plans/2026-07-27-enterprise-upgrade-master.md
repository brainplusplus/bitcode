# Enterprise Component Upgrade — Master Design Document

**Date:** 2026-07-27
**Scope:** Upgrade all 103 Stencil Web Components to enterprise-grade
**Philosophy:** Simple usage (sensible defaults, zero-config works), Complete capability (handles all edge cases)
**Approach:** Hybrid — component-level props for standalone use, FormEngine as optional orchestrator

---

## PHASE BREAKDOWN

| Phase | Doc | Scope | Depends On |
|-------|-----|-------|------------|
| 1 | `phase-1-core-infrastructure.md` | BcSetup, data-fetcher, field-utils, types, validation engine | — |
| 2 | `phase-2-field-components.md` | 34 field components — universal props/events/methods | Phase 1 |
| 3 | `phase-3-select-data-fields.md` | 6 select-family — 4-level data, cascading, dependent | Phase 1, 2 |
| 4 | `phase-4-datatable.md` | bc-datatable — 4-level data, enterprise features | Phase 1 |
| 5 | `phase-5-charts.md` | 11 chart components — 4-level data, events, methods | Phase 1 |
| 6 | `phase-6-reactivity-advanced.md` | BcSetup.reactivity(), cross-field, server-side validation | Phase 1, 2, 3 |

Each phase has its own design doc + implementation plan.
This master doc contains the full reference for all decisions.

## DOCUMENTATION STRUCTURE

All component docs live inside `packages/components/` — self-contained for future repo split.
Design/planning docs stay in root `docs/plans/` (they belong to the monorepo, not the component package).

```
docs/plans/                                    # Design & planning (monorepo-level)
├── 2026-07-27-enterprise-upgrade-master.md    # This file
├── 2026-07-27-phase-1-core-infrastructure.md
├── 2026-07-27-phase-2-field-components.md
├── ... (phase docs)

packages/components/                           # Self-contained component package
├── docs/                                      # ALL component documentation here
│   ├── README.md                              # Master index — links to everything
│   ├── getting-started.md                     # Quick start, install, standalone usage
│   ├── theming.md                             # Theme system guide (light/dark/custom)
│   ├── data-fetching.md                       # 4-level data strategy guide
│   ├── validation.md                          # 3-level validation guide
│   ├── reactivity.md                          # Dependent fields, cascading guide
│   ├── bc-setup.md                            # BcSetup API reference
│   ├── fields/
│   │   ├── bc-field-string.md
│   │   ├── bc-field-integer.md
│   │   ├── bc-field-select.md
│   │   ├── bc-field-date.md
│   │   └── ... (1 md per field, 34 total)
│   ├── datatable/
│   │   └── bc-datatable.md
│   ├── charts/
│   │   ├── bc-chart-bar.md
│   │   └── ... (1 md per chart, 11 total)
│   ├── layout/
│   │   └── ... (1 md per layout component, 10 total)
│   ├── dialogs/
│   │   └── ... (1 md per dialog, 4 total)
│   ├── widgets/
│   │   └── ... (1 md per widget)
│   └── utility/
│       └── ... (1 md per utility component)
├── src/
│   ├── core/
│   ├── components/
│   └── ...
```

### Component Doc Template (each `bc-*.md`):

```markdown
# bc-field-string

> Single-line text input with validation, formatting, and enterprise features.

## Quick Start
  (minimal working example — standalone HTML, no framework)

## Props
  (table: name, type, default, description — EVERY prop documented)

## Events
  (table: name, payload type, when emitted, example listener code)

## Methods
  (table: name, signature, returns, description, example call)

## Validation
  (built-in rules, custom validator, server-side — with examples)

## Data Binding / Reactivity
  (dependOn, dataSource — if applicable)

## Theming / CSS Custom Properties
  (which --bc-* vars affect this component, how to override)

## Examples
  ### Basic (standalone HTML)
  ### With Validation
  ### In a Form (with other fields)
  ### Server-side Data (if applicable)
  ### Custom Theme
  ### Framework Integration (React, Vue, Angular)

## Edge Cases & Notes

## Changelog
```

### When docs are generated:
- Phase 1 complete → `packages/components/docs/README.md`, `bc-setup.md`, `theming.md`, `data-fetching.md`, `validation.md`
- Phase 2 complete → `packages/components/docs/fields/*.md` (34 files)
- Phase 3 complete → update select-family docs with data features
- Phase 4 complete → `packages/components/docs/datatable/bc-datatable.md`
- Phase 5 complete → `packages/components/docs/charts/*.md` (11 files)
- Phase 6 complete → update `reactivity.md` + affected component docs

## MANDATORY DOC UPDATES (per phase checklist)

**Rule from AGENTS.md: "if you changed code, you changed docs. No exceptions."**

### After EVERY phase, update these:

| Doc | What to Update |
|-----|---------------|
| `AGENTS.md` | Component count, file structure rules (new core files), "Adding/Modifying Components" section (add docs step) |
| `docs/architecture.md` | Section "7. Web Components" — add BcSetup, theming, 4-level data, standalone note |
| `docs/codebase.md` | Section "Packages/Components" — add every new file/folder created |
| `docs/features.md` | Section "5. Form & UI Builder" — update feature status, add new features |
| `README.md` | Component count, standalone mention, theming mention |
| `packages/components/docs/README.md` | Master index — link to all component docs |

### Phase-specific doc updates:

| Phase | Additional Docs |
|-------|----------------|
| Phase 1 | `AGENTS.md` file structure (new core files), `docs/codebase.md` (new files), `docs/architecture.md` (BcSetup, theming, data-fetcher) |
| Phase 2 | `docs/features.md` (enterprise fields status), `AGENTS.md` (component count, shadow DOM change note) |
| Phase 3 | `docs/features.md` (select/data-driven status), `docs/architecture.md` (4-level data in component section) |
| Phase 4 | `docs/features.md` (datatable status) |
| Phase 5 | `docs/features.md` (charts status) |
| Phase 6 | `docs/features.md` (reactivity status), `docs/architecture.md` (reactivity system) |

### Docs that do NOT need updating:
- `CLAUDE.md` — just points to AGENTS.md
- `engine/docs/*` — engine is not changing
- `engine/docs/architecture.md` — engine architecture unchanged
- `engine/docs/codebase.md` — engine codebase unchanged

---

## 0. CORE PRINCIPLE: STANDALONE-FIRST

**Components MUST work anywhere — not just inside BitCode.**

This means:
1. **ZERO framework dependency** — every component works in plain HTML, React, Vue, Angular, or any framework
2. **`getApiClient()` is OPTIONAL** — components that need data use `dataSource` prop (plain URL) + native `fetch()`. If BitCode's `getApiClient()` is available, use it as enhancement, never as requirement
3. **`options` accepts BOTH JSON string AND JS object** — `options='[{"label":"A","value":"1"}]'` from HTML, or `:options="myArray"` from framework
4. **Reactivity via DOM events** — `dependOn` listens for standard `lcFieldChange` CustomEvents on the document. Works in any context where sibling components emit these events
5. **No magic globals** — no `window.__lc_*` required. Everything configurable via props
6. **Web Component standard** — works with `<script>` tag include, no build step required for consumers

### Standalone usage example (plain HTML, no BitCode):
```html
<script type="module" src="https://cdn.example.com/bc-components/bc-components.esm.js"></script>

<bc-field-string name="email" label="Email" required clearable hint="We'll never share your email" />
<bc-field-select name="country" label="Country" options='[{"label":"Indonesia","value":"ID"}]' searchable />
<bc-field-select name="city" label="City" depend-on="country" data-source="https://api.example.com/cities?country={country}" />
```

### BitCode-enhanced usage (automatic, zero config):
When used inside BitCode's `bc-view-form`, FormEngine auto-wires `depends_on`, `readonly_if`, `mandatory_if`, `formula` from model JSON. Components don't need to know about this — FormEngine calls their public methods (`setValue`, `setError`, etc).

---

## 1. AUDIT SUMMARY

### Current State
- 34 field components: most have only `name, label, value, placeholder, required, readonly, disabled`
- No validation state tracking (dirty/touched/pristine)
- No accessibility attributes (ARIA)
- No reactive/dependent field support
- No universal methods (validate, reset, focus, blur)
- No universal events beyond `lcFieldChange`
- 11 chart components: no events, minimal methods
- 1 datatable: functional but missing inline edit, column resize, virtual scroll
- Select/Link/Tags: hardcoded to `getApiClient()` — breaks standalone use
- 6 field components tightly coupled to `getApiClient()`: link, dynlink, tags, tableselect, image, file

### Target State
Every component has complete enterprise capability with zero-config defaults.
Developer writes `<bc-field-string name="email" label="Email" required />` and gets:
- Auto validation on blur
- Dirty/touched tracking
- ARIA attributes auto-generated
- Error display auto-managed
- Focus/blur events emitted
- validate()/reset()/clear() methods available
- Works in plain HTML without any framework

---

## 2. UNIVERSAL FIELD PROPS (all 34 field components)

### 2A. Props to ADD to ALL fields

```typescript
// --- Validation & State ---
@Prop() validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
@Prop() validationMessage: string = '';
@Prop() hint: string = '';                    // Helper text below field
@Prop() pattern: string = '';                 // Regex validation pattern
@Prop() patternMessage: string = '';          // Custom pattern error message

// --- UX ---
@Prop() size: 'sm' | 'md' | 'lg' = 'md';
@Prop() clearable: boolean = false;
@Prop() prefix: string = '';                  // Visual prefix text/icon
@Prop() suffix: string = '';                  // Visual suffix text/icon
@Prop() tooltip: string = '';                 // Tooltip on hover
@Prop() loading: boolean = false;             // Loading indicator
@Prop() autofocus: boolean = false;           // Focus on mount
@Prop() defaultValue: string = '';            // For reset()

// --- Reactivity ---
@Prop() dependOn: string = '';                // Parent field name
@Prop() dataSource: string = '';              // API endpoint with {field} placeholders
```

### 2B. Props to ADD to TEXT-BASED fields only
(string, text, smalltext, password, richtext, markdown, html, code, json, email-like)

```typescript
@Prop() minLength: number = 0;
@Prop() maxLength: number = 0;               // 0 = unlimited
@Prop() showCount: boolean = false;           // Character counter
```

### 2C. Props to ADD to NUMERIC fields only
(integer, float, decimal, currency, percent)

Already have `min`/`max` on some. Normalize:
```typescript
@Prop() min: number = 0;                     // Already on integer, percent
@Prop() max: number = 0;                     // Already on integer, percent, string
@Prop() step: number = 1;                    // Already on integer
```

### 2D. Props MISSING per component (specific gaps)

| Component | Missing Props to Add |
|-----------|---------------------|
| bc-field-checkbox | `required` |
| bc-field-toggle | `required`, `readonly` |
| bc-field-color | `required`, `readonly` |
| bc-field-rating | `required`, `readonly` |
| bc-field-barcode | `required`, `readonly`, `placeholder` |
| bc-field-duration | `required`, `readonly`, `placeholder` |
| bc-field-geo | `required`, `readonly`, `placeholder` |
| bc-field-signature | `required`, `readonly` |
| bc-field-radio | `required`, `readonly` |

### 2E. Internal State (auto-managed)

```typescript
@State() private _dirty: boolean = false;
@State() private _touched: boolean = false;
@State() private _errors: string[] = [];
@State() private _initialValue: any;          // Set on componentWillLoad
```

---

## 3. UNIVERSAL FIELD EVENTS (all 34 field components)

### Current: only `lcFieldChange`

### Add:

| Event | Payload Type | When Emitted |
|-------|-------------|--------------|
| `lcFieldFocus` | `{name: string, value: unknown}` | Field receives focus |
| `lcFieldBlur` | `{name: string, value: unknown, dirty: boolean, touched: boolean}` | Field loses focus, auto-validate triggers here |
| `lcFieldClear` | `{name: string, oldValue: unknown}` | Clear button clicked or clear() called |
| `lcFieldInvalid` | `{name: string, value: unknown, errors: string[]}` | Validation fails |
| `lcFieldValid` | `{name: string, value: unknown}` | Validation passes |

---

## 4. UNIVERSAL FIELD METHODS (all 34 field components)

```typescript
@Method() async validate(): Promise<{valid: boolean, errors: string[]}> { ... }
@Method() async reset(): Promise<void> { ... }        // Reset to defaultValue or initialValue
@Method() async clear(): Promise<void> { ... }        // Set to empty
@Method() async setValue(value: any, emit?: boolean): Promise<void> { ... }
@Method() async getValue(): Promise<any> { ... }
@Method() async focus(): Promise<void> { ... }
@Method() async blur(): Promise<void> { ... }
@Method() async isDirty(): Promise<boolean> { ... }
@Method() async isTouched(): Promise<boolean> { ... }
@Method() async setError(message: string): Promise<void> { ... }
@Method() async clearError(): Promise<void> { ... }
```

---

## 5. DATA-DRIVEN COMPONENTS — HYBRID 4-LEVEL DATA STRATEGY

Applies to: bc-field-select, bc-field-link, bc-field-dynlink, bc-field-tags, bc-field-tableselect, bc-field-multicheck, bc-datatable, all charts.

**All 4 levels available simultaneously. Developer picks what fits. Can combine.**

### 5.0. The 4 Levels

#### Level 1: Local Data (zero fetch, data in prop)
```html
<bc-field-select name="status" options='[{"label":"Active","value":"active"}]' />
<bc-datatable columns='[...]' data='[{"id":1,"name":"John"}]' />
```

#### Level 2: URL Endpoint (auto-fetch with native fetch())
```html
<bc-field-select name="city" data-source="/api/cities" />
<bc-datatable columns='[...]' data-source="/api/users" server-side />
```

#### Level 3: URL + Event Intercept (modify request/response)
```html
<bc-datatable id="t" data-source="/api/users" server-side />
<script>
document.getElementById('t').addEventListener('lcBeforeFetch', (e) => {
  e.detail.headers['X-Custom'] = 'value';
});
document.getElementById('t').addEventListener('lcAfterFetch', (e) => {
  e.detail.data = e.detail.response.items;
  e.detail.total = e.detail.response.totalCount;
});
</script>
```

#### Level 4: Custom Fetcher Function (full control via JS property)
```html
<bc-datatable id="t" columns='[...]' />
<script>
document.getElementById('t').dataFetcher = async (params) => {
  // params = { page, pageSize, sort, filters, search }
  // Fetch anywhere, parse anything, return standard format
  const res = await fetch('https://any-api.com/data', { ... });
  const json = await res.json();
  return { data: json.records, total: json.count };
};

// Same for select
document.getElementById('citySelect').optionsFetcher = async (query, params) => {
  // query = search text, params = { dependValues: { province: 'jabar' } }
  return [{ label: 'Bandung', value: 'bdg' }];
};
</script>
```

#### Resolution priority (inside component):
```
1. dataFetcher/optionsFetcher (JS property)  → Level 4
2. lcBeforeFetch/lcAfterFetch listeners      → Level 3 (with dataSource)
3. dataSource prop                           → Level 2 (native fetch)
4. data/options prop                         → Level 1 (local)
5. model prop + getApiClient available       → BitCode fallback
6. nothing                                   → empty state
```

### 5A. SELECT-FAMILY Props to ADD
(bc-field-select, bc-field-link, bc-field-dynlink, bc-field-tags, bc-field-tableselect, bc-field-multicheck, bc-field-radio)

```typescript
// --- Data Source (all 4 levels) ---
@Prop() dataSource: string = '';             // Level 2: URL endpoint
@Prop() serverSide: boolean = false;         // Enable server-side search
@Prop() searchable: boolean = true;          // Enable search/filter
@Prop() multiple: boolean = false;           // Multi-select (for bc-field-select)
@Prop() displayField: string = 'name';       // Which field to display
@Prop() valueField: string = 'id';           // Which field as value
@Prop() groupBy: string = '';                // Group options by field
@Prop() creatable: boolean = false;          // Allow creating new options
@Prop() pageSize: number = 50;              // API page size for server-side
@Prop() debounceMs: number = 300;            // Debounce for search
@Prop() minSearchLength: number = 1;         // Min chars before search
@Prop() noResultsText: string = '';          // Custom "no results" text
@Prop() loadingText: string = '';            // Custom loading text
@Prop() fetchHeaders: string = '';           // Custom headers JSON for fetch

// Level 4: JS property (not @Prop, set via JS)
// element.optionsFetcher = async (query, params) => [...]
```

### 5B. SELECT-FAMILY Events to ADD

| Event | Payload | When |
|-------|---------|------|
| `lcBeforeFetch` | `{url, headers, params}` | Before fetch — modify request (Level 3) |
| `lcAfterFetch` | `{response, data, total}` | After fetch — transform response (Level 3) |
| `lcOptionsLoad` | `{options: any[], total: number}` | Options loaded (any level) |
| `lcOptionsError` | `{error: string}` | Options load failed |
| `lcOptionCreate` | `{value: string}` | New option created (creatable) |
| `lcDropdownOpen` | `{}` | Dropdown opened |
| `lcDropdownClose` | `{}` | Dropdown closed |

### 5C. SELECT-FAMILY Methods to ADD

```typescript
@Method() async loadOptions(query?: string): Promise<void> { ... }
@Method() async reloadOptions(): Promise<void> { ... }
@Method() async getOptions(): Promise<any[]> { ... }
@Method() async getSelectedOptions(): Promise<any[]> { ... }
@Method() async setOptions(options: any[]): Promise<void> { ... }  // Programmatic set
@Method() async open(): Promise<void> { ... }
@Method() async close(): Promise<void> { ... }
```

---

## 6. DATATABLE ENHANCEMENTS (bc-datatable)

Uses same 4-level data strategy from Section 5.0.

### 6A. Props to ADD

```typescript
// --- Data (4 levels) ---
@Prop() data: string = '[]';                // Level 1: Local data
@Prop() dataSource: string = '';             // Level 2: URL endpoint
// Level 3: lcBeforeFetch/lcAfterFetch events
// Level 4: element.dataFetcher = async (params) => { data, total }
@Prop() fetchHeaders: string = '';           // Custom headers JSON for fetch

// --- Features ---
@Prop() editable: boolean = false;           // Enable inline editing
@Prop() expandable: boolean = false;         // Row expansion
@Prop() resizableColumns: boolean = false;   // Column resize
@Prop() virtualScroll: boolean = false;      // Virtual scrolling for large data
@Prop() stickyHeader: boolean = true;        // Sticky header
@Prop() rowHeight: number = 40;             // Row height for virtual scroll
@Prop() emptyText: string = '';             // Custom empty state text
@Prop() loading: boolean = false;            // External loading control
```

### 6B. Events to ADD

| Event | Payload | When |
|-------|---------|------|
| `lcBeforeFetch` | `{url, headers, params}` | Before fetch — modify request (Level 3) |
| `lcAfterFetch` | `{response, data, total}` | After fetch — transform response (Level 3) |
| `lcRowEdit` | `{row, field, value, oldValue}` | Inline cell edited |
| `lcRowExpand` | `{row, expanded}` | Row expanded/collapsed |
| `lcColumnResize` | `{column, width}` | Column resized |
| `lcColumnReorder` | `{columns}` | Columns reordered |
| `lcCellClick` | `{row, column, value}` | Cell clicked |
| `lcScrollEnd` | `{direction}` | Scrolled to bottom (infinite scroll) |
| `lcFilterChange` | `{filters}` | Filter changed |
| `lcSortChange` | `{sorts}` | Sort changed |
| `lcPageChange` | `{page, pageSize}` | Page changed |

### 6C. Methods to ADD

```typescript
@Method() async refresh(): Promise<void> { ... }
@Method() async getData(): Promise<any[]> { ... }
@Method() async setData(data: any[]): Promise<void> { ... }  // Programmatic set
@Method() async getSelected(): Promise<any[]> { ... }
@Method() async clearSelection(): Promise<void> { ... }
@Method() async selectAll(): Promise<void> { ... }
@Method() async expandRow(id: string): Promise<void> { ... }
@Method() async collapseRow(id: string): Promise<void> { ... }
@Method() async goToPage(page: number): Promise<void> { ... }
@Method() async sortBy(column: string, direction: 'asc' | 'desc'): Promise<void> { ... }
@Method() async exportCSV(options?: any): Promise<void> { ... }
@Method() async exportPDF(options?: any): Promise<void> { ... }
@Method() async scrollToRow(id: string): Promise<void> { ... }
```

---

## 7. CHART ENHANCEMENTS (all 11 chart components)

Uses same 4-level data strategy from Section 5.0.

### 7A. Props to ADD (all charts)

```typescript
// --- Data (4 levels) ---
// data prop already exists (Level 1)
@Prop() dataSource: string = '';             // Level 2: API endpoint
// Level 3: lcBeforeFetch/lcAfterFetch events
// Level 4: element.dataFetcher = async () => [...]
@Prop() fetchHeaders: string = '';           // Custom headers JSON
@Prop() refreshInterval: number = 0;         // Auto-refresh interval (ms), 0=disabled

// --- Display ---
@Prop() colors: string = '';                 // Custom color palette JSON
@Prop() legend: boolean = true;              // Show legend
@Prop() tooltipEnabled: boolean = true;      // Show tooltip
@Prop() animate: boolean = true;             // Enable animation
@Prop() height: string = '300px';            // Chart height
@Prop() loading: boolean = false;            // Loading state
```

### 7B. Events to ADD (all charts)

| Event | Payload | When |
|-------|---------|------|
| `lcBeforeFetch` | `{url, headers}` | Before fetch (Level 3) |
| `lcAfterFetch` | `{response, data}` | After fetch (Level 3) |
| `lcChartClick` | `{name, value, dataIndex}` | Data point clicked |
| `lcChartHover` | `{name, value, dataIndex}` | Data point hovered |

### 7C. Methods to ADD (all charts)

```typescript
@Method() async updateData(data: any): Promise<void> { ... }
@Method() async setData(data: any): Promise<void> { ... }    // Alias, programmatic
@Method() async exportImage(format?: string): Promise<string> { ... }
@Method() async refresh(): Promise<void> { ... }
@Method() async resize(): Promise<void> { ... }
```

---

## 8. REACTIVITY SYSTEM (Dependent Fields)

### How it works (standalone, no framework required):

```html
<!-- Province → City → District cascading — works in plain HTML -->
<bc-field-select name="province" options='[{"label":"Jawa Barat","value":"jabar"}]' />
<bc-field-select name="city" depend-on="province" data-source="/api/cities?province={province}" />
<bc-field-select name="district" depend-on="city" data-source="/api/districts?city={city}" />
```

### Implementation:
1. Component with `dependOn` adds a document-level listener for `lcFieldChange` CustomEvent on `connectedCallback()`
2. Listener checks `event.detail.name === this.dependOn` (or matches one of comma-separated names)
3. On match: replaces `{fieldName}` placeholders in `dataSource` with the new value
4. Fetches new options via **native `fetch()`** — NOT `getApiClient()`
5. Clears current value (child resets when parent changes)
6. Emits own `lcFieldChange` to cascade further down the chain
7. Removes listener on `disconnectedCallback()`

### Data fetching strategy (standalone-first):
```
dataSource prop provided?
  YES → use native fetch(resolvedUrl)
        → expect response: { data: [...] } or plain array [...]
        → auto-detect response shape
  NO  → use options prop (local data)
        → if model prop provided AND getApiClient exists → fallback to api-client (BitCode mode)
        → if model prop provided AND no api-client → no-op, log warning
```

This means:
- **Plain HTML**: `data-source="/api/cities?province={province}"` — works with any REST API
- **BitCode**: `model="city"` — auto-uses `getApiClient().search()` if available
- **Both**: `model="city" data-source="/custom/api"` — dataSource takes priority

### Edge cases handled:
- Parent not yet rendered → listener waits for event, no error
- API fails → component shows error state via `validationStatus='invalid'`, keeps old options
- Circular dependency → max depth 10, then stops with console warning
- Multiple parents → `dependOn` supports comma-separated: `depend-on="province,type"`
- URL template → supports `{fieldName}` anywhere in dataSource URL
- Response format → auto-detects `{data:[...]}` or plain `[...]` or `{results:[...]}`
- CORS → developer's responsibility, component just calls fetch()
- Auth headers → optional `fetchHeaders` prop for custom headers, or global `window.__bc_fetch_headers`

### FormEngine integration (optional, BitCode-only):
FormEngine can also set values/options programmatically via `setValue()` and component methods.
The `depends_on`, `readonly_if`, `mandatory_if`, `formula` in FormEngine continue to work.
Component-level `dependOn` is the standalone equivalent — both can coexist.

---

## 9. TYPES UPDATE (core/types.ts)

```typescript
// Add to FieldChangeEvent
export interface FieldChangeEvent {
  name: string;
  value: unknown;
  oldValue: unknown;
}

// New event types
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

export interface FieldValidResult {
  name: string;
  value: unknown;
}

// Chart events
export interface ChartClickEvent {
  name: string;
  value: unknown;
  dataIndex: number;
}

// DataTable events
export interface RowEditEvent {
  row: Record<string, unknown>;
  field: string;
  value: unknown;
  oldValue: unknown;
}

export interface ColumnResizeEvent {
  column: string;
  width: number;
}

export interface CellClickEvent {
  row: Record<string, unknown>;
  column: string;
  value: unknown;
}

// Select events
export interface OptionsLoadEvent {
  options: unknown[];
  total: number;
}

export interface OptionCreateEvent {
  value: string;
}
```

---

## 10. IMPLEMENTATION STRATEGY

### Approach: Shared Utility Modules
Create shared utility modules (NOT base classes — Stencil doesn't support class inheritance well):

### File: `packages/components/src/core/field-utils.ts`
Common field logic:
```typescript
export function validateField(value, opts: {required, minLength, maxLength, pattern, patternMessage}): {valid, errors}
export function renderFieldWrapper(props, content, extras): JSX
export function getAriaAttrs(props): Record<string, string>
export function trackDirtyTouched(currentValue, initialValue): {dirty, touched}
```

### File: `packages/components/src/core/data-fetcher.ts`
4-level data fetching (standalone, no BitCode dependency):
```typescript
interface FetchParams {
  page?: number;
  pageSize?: number;
  sort?: Array<{field: string, direction: string}>;
  filters?: any;
  search?: string;
  dependValues?: Record<string, unknown>;  // For cascading
}

interface FetchResult {
  data: any[];
  total: number;
}

// Resolves which level to use and fetches data
export async function fetchData(opts: {
  fetcher?: (params: FetchParams) => Promise<FetchResult>;  // Level 4
  element?: HTMLElement;                                      // Level 3 (events)
  dataSource?: string;                                        // Level 2
  localData?: string;                                         // Level 1
  model?: string;                                             // BitCode fallback
  fetchHeaders?: string;                                      // Custom headers
  params?: FetchParams;
}): Promise<FetchResult>

// For select-family: resolves options
export async function fetchOptions(opts: {
  fetcher?: (query: string, params: FetchParams) => Promise<any[]>;  // Level 4
  element?: HTMLElement;                                               // Level 3
  dataSource?: string;                                                 // Level 2
  localOptions?: string;                                               // Level 1
  model?: string;                                                      // BitCode fallback
  query?: string;
  fetchHeaders?: string;
  params?: FetchParams;
}): Promise<any[]>

// Auto-detect response format
export function normalizeResponse(response: any): FetchResult
// Handles: { data: [...] }, [...], { results: [...] }, { items: [...] }, { records: [...] }

// Resolve URL template placeholders
export function resolveUrl(template: string, values: Record<string, unknown>): string
// "/api/cities?province={province}" + { province: "jabar" } → "/api/cities?province=jabar"
```

### Order of implementation:
1. `core/types.ts` — Add new event interfaces
2. `core/field-utils.ts` — Create shared field utilities
3. `core/data-fetcher.ts` — Create 4-level data fetching utility
4. All 34 field components — Add universal props/events/methods
5. Select-family components (6) — Add select-specific + 4-level data
6. bc-datatable — Add datatable enhancements + 4-level data
7. All 11 chart components — Add chart enhancements + 4-level data
8. Build & verify (`npm run build` in packages/components/)
9. Update docs

---

## 11. COMPONENT-BY-COMPONENT CHANGE MATRIX

### FIELD Components (34) — Universal additions apply to ALL

| Component | Specific Additional Props | Notes |
|-----------|--------------------------|-------|
| bc-field-string | (none extra) | Already has max, widget |
| bc-field-text | minLength, maxLength, showCount, rows (has) | Textarea |
| bc-field-smalltext | minLength, maxLength, showCount | Textarea |
| bc-field-password | minLength, maxLength, showReveal:boolean | Add show/hide toggle |
| bc-field-integer | min (has), max (has), step (has) | Complete |
| bc-field-float | min, max, step | Add min/max/step |
| bc-field-decimal | min, max, step | Add min/max/step |
| bc-field-currency | min, max, step | Add min/max/step |
| bc-field-percent | min (has), max (has), step | Add step |
| bc-field-date | min, max | Date range |
| bc-field-datetime | min, max | DateTime range |
| bc-field-time | min, max, step | Time range |
| bc-field-checkbox | required | Missing |
| bc-field-toggle | required, readonly | Missing |
| bc-field-color | required, readonly | Missing |
| bc-field-rating | required, readonly | Missing |
| bc-field-radio | required, readonly | Missing |
| bc-field-barcode | required, readonly, placeholder | Missing |
| bc-field-duration | required, readonly, placeholder | Missing |
| bc-field-geo | required, readonly, placeholder | Missing |
| bc-field-signature | required, readonly | Missing |
| bc-field-select | searchable, multiple, serverSide, valueField, displayField, groupBy, creatable, pageSize, debounceMs | Major upgrade |
| bc-field-link | (select-specific already partially there) | Enhance |
| bc-field-dynlink | (select-specific) | Enhance |
| bc-field-tags | (select-specific) | Enhance |
| bc-field-tableselect | (select-specific) | Enhance |
| bc-field-multicheck | (select-specific subset) | Enhance |
| bc-field-richtext | minLength, maxLength, showCount | Text-based |
| bc-field-markdown | minLength, maxLength, showCount | Text-based |
| bc-field-html | minLength, maxLength, showCount | Text-based |
| bc-field-code | minLength, maxLength, showCount | Text-based |
| bc-field-json | (none extra) | Code-based |
| bc-field-file | (already rich) | Minor additions |
| bc-field-image | (already rich) | Minor additions |

---

## 12. WHAT IS NOT IN SCOPE

These are explicitly excluded to keep scope manageable:

- **Business logic props** (approvalRequired, auditLog, versioning, workflowState) — these belong in FormEngine/backend, not component level
- **Security props** (encrypt, mask, sensitive) — backend concern
- **Mobile-specific props** (mobileBreakpoint, mobileVariant, swipeEnabled) — CSS handles responsive
- **Advanced accessibility** (ariaLive, role, inputMode) — auto-generated from existing props
- **Performance props** (virtualScroll for selects, cacheOptions, throttleMs) — implement internally with sensible defaults, no prop needed
- **DOM methods** (getElement, getBoundingClientRect, scrollIntoView) — native DOM already provides these

---

**Generated by AI Assistant**
**Date: 2026-07-27**
