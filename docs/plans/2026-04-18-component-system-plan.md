# Component System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a complete 86-component Stencil Web Components library for the low-code ERP platform, with dual-mode rendering (JSON definition + HTML custom elements).

**Architecture:** Stencil monorepo at `packages/components/` compiles to lazy-loaded JS bundles. Go engine's ViewRenderer is rewritten to output `<lc-*>` custom elements instead of raw HTML. A client-side form-engine handles reactive behaviors (depends_on, formula, etc.). Components communicate via CustomEvents and talk to the Go engine via an API client.

**Tech Stack:** Stencil (Web Components compiler), TypeScript, Go 1.21+, ECharts (charts), TipTap (rich text), CodeMirror 6 (code editor), Leaflet (maps), SortableJS (drag-drop), FullCalendar (calendar view)

**Design Doc:** `docs/plans/2026-04-18-component-system-design.md`

---

## Batch 1: Foundation (Tasks 1-5)

### Task 1: Scaffold Stencil Project

**Files:**
- Create: `packages/components/package.json`
- Create: `packages/components/stencil.config.ts`
- Create: `packages/components/tsconfig.json`
- Create: `packages/components/src/global/global.css`
- Create: `packages/components/src/core/types.ts`

**Step 1: Initialize Stencil project**

```bash
cd packages
npm init stencil component components
cd components
```

**Step 2: Configure stencil.config.ts**

```typescript
import { Config } from '@stencil/core';

export const config: Config = {
  namespace: 'lc-components',
  outputTargets: [
    {
      type: 'dist',
      esmLoaderPath: '../loader',
    },
    {
      type: 'dist-custom-elements',
    },
    {
      type: 'www',
      serviceWorker: null,
    },
  ],
  globalStyle: 'src/global/global.css',
};
```

**Step 3: Create design tokens in global.css**

```css
:root {
  --lc-primary: #4f46e5;
  --lc-primary-hover: #4338ca;
  --lc-success: #10b981;
  --lc-warning: #f59e0b;
  --lc-danger: #ef4444;
  --lc-info: #3b82f6;
  --lc-muted: #6b7280;

  --lc-bg: #ffffff;
  --lc-bg-secondary: #f9fafb;
  --lc-bg-hover: #f3f4f6;
  --lc-border-color: #e5e7eb;
  --lc-text: #111827;
  --lc-text-secondary: #6b7280;

  --lc-font-family: 'Inter', system-ui, -apple-system, sans-serif;
  --lc-font-size-xs: 0.75rem;
  --lc-font-size-sm: 0.875rem;
  --lc-font-size-base: 1rem;
  --lc-font-size-lg: 1.125rem;
  --lc-font-size-xl: 1.25rem;

  --lc-spacing-xs: 0.25rem;
  --lc-spacing-sm: 0.5rem;
  --lc-spacing-md: 1rem;
  --lc-spacing-lg: 1.5rem;
  --lc-spacing-xl: 2rem;

  --lc-radius-sm: 0.25rem;
  --lc-radius-md: 0.375rem;
  --lc-radius-lg: 0.5rem;
  --lc-radius-full: 9999px;

  --lc-shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --lc-shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  --lc-shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1);

  --lc-transition: 150ms ease;
}
```

**Step 4: Create shared types**

```typescript
// src/core/types.ts
export interface FieldChangeEvent {
  name: string;
  value: any;
  oldValue: any;
}

export interface FormSubmitEvent {
  model: string;
  data: Record<string, any>;
  id?: string;
}

export interface ActionClickEvent {
  process: string;
  recordId?: string;
}

export interface ListParams {
  page?: number;
  pageSize?: number;
  sort?: string;
  order?: 'asc' | 'desc';
  filters?: Record<string, any>;
  q?: string;
}

export interface ListResponse {
  data: Record<string, any>[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export type FieldType =
  | 'string' | 'smalltext' | 'text' | 'richtext' | 'markdown' | 'html' | 'code' | 'password'
  | 'integer' | 'float' | 'decimal' | 'currency' | 'percent'
  | 'boolean' | 'toggle' | 'selection' | 'radio'
  | 'many2one' | 'one2many' | 'many2many' | 'dynamic_link' | 'many2many_check' | 'table_multiselect'
  | 'date' | 'time' | 'datetime' | 'duration'
  | 'file' | 'image' | 'signature'
  | 'barcode' | 'color' | 'geolocation' | 'rating' | 'json'
  | 'computed';

export type WidgetType =
  | 'statusbar' | 'priority' | 'handle' | 'badge' | 'copy'
  | 'phone' | 'email' | 'url' | 'progress' | 'domain';
```

**Step 5: Verify build works**

Run: `npm run build`
Expected: Build succeeds, dist/ directory created

**Step 6: Commit**

```bash
git add packages/
git commit -m "feat: scaffold Stencil component library with design tokens and shared types"
```

---

### Task 2: Core Utilities — API Client, Event Bus, Expression Evaluator

**Files:**
- Create: `packages/components/src/core/api-client.ts`
- Create: `packages/components/src/core/event-bus.ts`
- Create: `packages/components/src/utils/expression-eval.ts`
- Create: `packages/components/src/utils/format.ts`
- Create: `packages/components/src/utils/validators.ts`
- Test: `packages/components/src/utils/expression-eval.spec.ts`
- Test: `packages/components/src/utils/format.spec.ts`

**Step 1: Write expression evaluator tests**

```typescript
// src/utils/expression-eval.spec.ts
import { evaluate } from './expression-eval';

describe('expression-eval', () => {
  it('evaluates simple comparison', () => {
    expect(evaluate("status == 'draft'", { status: 'draft' })).toBe(true);
    expect(evaluate("status == 'draft'", { status: 'confirmed' })).toBe(false);
  });

  it('evaluates arithmetic', () => {
    expect(evaluate('qty * unit_price', { qty: 5, unit_price: 100 })).toBe(500);
  });

  it('evaluates logical operators', () => {
    expect(evaluate("status == 'draft' && total > 0", { status: 'draft', total: 100 })).toBe(true);
  });

  it('evaluates nested field access', () => {
    expect(evaluate('contact_id.email', { contact_id: { email: 'a@b.com' } })).toBe('a@b.com');
  });

  it('returns false for undefined fields', () => {
    expect(evaluate("missing == 'x'", {})).toBe(false);
  });
});
```

**Step 2: Implement expression evaluator**

```typescript
// src/utils/expression-eval.ts
// Safe expression evaluator — NO eval() or Function()
// Supports: ==, !=, >, <, >=, <=, &&, ||, !, +, -, *, /, field.subfield

type Context = Record<string, any>;

function resolveField(path: string, ctx: Context): any {
  return path.split('.').reduce((obj, key) => obj?.[key], ctx);
}

// Tokenizer and recursive descent parser for safe evaluation
// (Full implementation — see design doc Section 6)
export function evaluate(expr: string, ctx: Context): any {
  // Implementation: tokenize -> parse -> evaluate AST
  // This is a safe sandboxed evaluator with no access to globals
}
```

**Step 3: Write format utility tests**

```typescript
// src/utils/format.spec.ts
import { formatCurrency, formatDate, formatNumber, formatPercent } from './format';

describe('format', () => {
  it('formats currency', () => {
    expect(formatCurrency(1000, 'USD')).toBe('$1,000.00');
    expect(formatCurrency(1000000, 'IDR')).toBe('Rp1.000.000');
  });

  it('formats date', () => {
    expect(formatDate('2026-04-18')).toBe('18 Apr 2026');
  });

  it('formats percent', () => {
    expect(formatPercent(75)).toBe('75%');
  });
});
```

**Step 4: Implement format utilities**

```typescript
// src/utils/format.ts
export function formatCurrency(value: number, currency: string = 'USD', precision: number = 2): string {
  return new Intl.NumberFormat(getCurrencyLocale(currency), {
    style: 'currency',
    currency,
    minimumFractionDigits: precision,
    maximumFractionDigits: precision,
  }).format(value);
}

export function formatDate(value: string, format: string = 'medium'): string {
  const date = new Date(value);
  return new Intl.DateTimeFormat('en-GB', { dateStyle: format as any }).format(date);
}

export function formatNumber(value: number, precision: number = 0): string {
  return new Intl.NumberFormat('en-US', {
    minimumFractionDigits: precision,
    maximumFractionDigits: precision,
  }).format(value);
}

export function formatPercent(value: number): string {
  return `${value}%`;
}

function getCurrencyLocale(currency: string): string {
  const map: Record<string, string> = { IDR: 'id-ID', USD: 'en-US', EUR: 'de-DE', GBP: 'en-GB' };
  return map[currency] || 'en-US';
}
```

**Step 5: Implement API client**

```typescript
// src/core/api-client.ts
import { ListParams, ListResponse } from './types';

export class LcApiClient {
  private baseUrl: string;

  constructor(baseUrl: string = '') {
    this.baseUrl = baseUrl;
  }

  async list(model: string, params?: ListParams): Promise<ListResponse> {
    const query = new URLSearchParams();
    if (params?.page) query.set('page', String(params.page));
    if (params?.pageSize) query.set('page_size', String(params.pageSize));
    if (params?.sort) query.set('sort', params.sort);
    if (params?.order) query.set('order', params.order);
    if (params?.q) query.set('q', params.q);
    const res = await fetch(`${this.baseUrl}/api/${model}s?${query}`);
    return res.json();
  }

  async read(model: string, id: string): Promise<Record<string, any>> {
    const res = await fetch(`${this.baseUrl}/api/${model}s/${id}`);
    return res.json();
  }

  async create(model: string, data: Record<string, any>): Promise<Record<string, any>> {
    const res = await fetch(`${this.baseUrl}/api/${model}s`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return res.json();
  }

  async update(model: string, id: string, data: Record<string, any>): Promise<Record<string, any>> {
    const res = await fetch(`${this.baseUrl}/api/${model}s/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return res.json();
  }

  async delete(model: string, id: string): Promise<void> {
    await fetch(`${this.baseUrl}/api/${model}s/${id}`, { method: 'DELETE' });
  }

  async action(model: string, id: string, action: string): Promise<Record<string, any>> {
    const res = await fetch(`${this.baseUrl}/api/${model}s/${id}/${action}`, { method: 'POST' });
    return res.json();
  }

  async upload(file: File): Promise<{ url: string }> {
    const form = new FormData();
    form.append('file', file);
    const res = await fetch(`${this.baseUrl}/api/upload`, { method: 'POST', body: form });
    return res.json();
  }
}

// Global singleton
let _client: LcApiClient | null = null;
export function getApiClient(): LcApiClient {
  if (!_client) {
    _client = new LcApiClient((window as any).__lc_base_url || '');
  }
  return _client;
}
```

**Step 6: Implement event bus**

```typescript
// src/core/event-bus.ts
type Handler = (data: any) => void;

class EventBus {
  private handlers: Map<string, Set<Handler>> = new Map();

  on(event: string, handler: Handler): () => void {
    if (!this.handlers.has(event)) {
      this.handlers.set(event, new Set());
    }
    this.handlers.get(event)!.add(handler);
    return () => this.handlers.get(event)?.delete(handler);
  }

  emit(event: string, data?: any): void {
    this.handlers.get(event)?.forEach(h => h(data));
  }
}

export const eventBus = new EventBus();
```

**Step 7: Implement validators**

```typescript
// src/utils/validators.ts
export function validateEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
}

export function validateUrl(value: string): boolean {
  try { new URL(value); return true; } catch { return false; }
}

export function validatePhone(value: string): boolean {
  return /^[+]?[\d\s()-]{7,20}$/.test(value);
}

export function validateRequired(value: any): boolean {
  if (value === null || value === undefined) return false;
  if (typeof value === 'string') return value.trim().length > 0;
  return true;
}

export function validateMax(value: string | number, max: number): boolean {
  if (typeof value === 'string') return value.length <= max;
  return value <= max;
}

export function validateMin(value: string | number, min: number): boolean {
  if (typeof value === 'string') return value.length >= min;
  return value >= min;
}
```

**Step 8: Run tests**

Run: `npm test`
Expected: All expression-eval and format tests pass

**Step 9: Commit**

```bash
git add packages/components/src/core/ packages/components/src/utils/
git commit -m "feat: add core utilities — API client, event bus, expression evaluator, formatters, validators"
```

---

### Task 3: Form Behavior Engine

**Files:**
- Create: `packages/components/src/core/form-engine.ts`
- Test: `packages/components/src/core/form-engine.spec.ts`

**Step 1: Write form engine tests**

```typescript
// src/core/form-engine.spec.ts
import { FormEngine } from './form-engine';

describe('FormEngine', () => {
  it('evaluates depends_on to show/hide fields', () => {
    const engine = new FormEngine({
      fields: {
        status: { type: 'selection' },
        lost_reason: { type: 'text', depends_on: "status == 'lost'" },
      }
    });
    engine.setValues({ status: 'draft' });
    expect(engine.isVisible('lost_reason')).toBe(false);
    engine.setValues({ status: 'lost' });
    expect(engine.isVisible('lost_reason')).toBe(true);
  });

  it('evaluates readonly_if', () => {
    const engine = new FormEngine({
      fields: {
        status: { type: 'selection' },
        name: { type: 'string', readonly_if: "status != 'draft'" },
      }
    });
    engine.setValues({ status: 'confirmed' });
    expect(engine.isReadonly('name')).toBe(true);
  });

  it('evaluates formula', () => {
    const engine = new FormEngine({
      fields: {
        qty: { type: 'integer' },
        price: { type: 'currency' },
        total: { type: 'currency', formula: 'qty * price' },
      }
    });
    engine.setValues({ qty: 5, price: 100 });
    expect(engine.getComputedValue('total')).toBe(500);
  });

  it('evaluates mandatory_if', () => {
    const engine = new FormEngine({
      fields: {
        type: { type: 'selection' },
        company_name: { type: 'string', mandatory_if: "type == 'company'" },
      }
    });
    engine.setValues({ type: 'company' });
    expect(engine.isMandatory('company_name')).toBe(true);
    engine.setValues({ type: 'person' });
    expect(engine.isMandatory('company_name')).toBe(false);
  });
});
```

**Step 2: Implement form engine**

```typescript
// src/core/form-engine.ts
import { evaluate } from '../utils/expression-eval';

interface FieldConfig {
  type: string;
  depends_on?: string;
  readonly_if?: string;
  mandatory_if?: string;
  fetch_from?: string;
  formula?: string;
  required?: boolean;
  readonly?: boolean;
}

interface FormConfig {
  fields: Record<string, FieldConfig>;
}

export class FormEngine {
  private config: FormConfig;
  private values: Record<string, any> = {};

  constructor(config: FormConfig) {
    this.config = config;
  }

  setValues(values: Record<string, any>): void {
    this.values = { ...this.values, ...values };
  }

  getValue(field: string): any {
    return this.values[field];
  }

  isVisible(field: string): boolean {
    const cfg = this.config.fields[field];
    if (!cfg?.depends_on) return true;
    return !!evaluate(cfg.depends_on, this.values);
  }

  isReadonly(field: string): boolean {
    const cfg = this.config.fields[field];
    if (cfg?.readonly) return true;
    if (!cfg?.readonly_if) return false;
    return !!evaluate(cfg.readonly_if, this.values);
  }

  isMandatory(field: string): boolean {
    const cfg = this.config.fields[field];
    if (cfg?.required) return true;
    if (!cfg?.mandatory_if) return false;
    return !!evaluate(cfg.mandatory_if, this.values);
  }

  getComputedValue(field: string): any {
    const cfg = this.config.fields[field];
    if (!cfg?.formula) return this.values[field];
    return evaluate(cfg.formula, this.values);
  }

  getFetchPath(field: string): string | null {
    return this.config.fields[field]?.fetch_from || null;
  }
}
```

**Step 3: Run tests**

Run: `npm test`
Expected: All form engine tests pass

**Step 4: Commit**

```bash
git add packages/components/src/core/form-engine*
git commit -m "feat: add form behavior engine with depends_on, readonly_if, mandatory_if, formula"
```

---

### Task 4: Extend Go Model Parser — New Field Types

**Files:**
- Modify: `engine/internal/compiler/parser/model.go`
- Modify: `engine/internal/compiler/parser/model_test.go`

**Step 1: Add new field type constants to model.go**

Add after existing constants (line 27):

```go
const (
    // ... existing constants unchanged ...

    // New field types
    FieldSmallText   FieldType = "smalltext"
    FieldRichText    FieldType = "richtext"
    FieldMarkdown    FieldType = "markdown"
    FieldHTML        FieldType = "html"
    FieldCode        FieldType = "code"
    FieldPassword    FieldType = "password"
    FieldFloat       FieldType = "float"
    FieldCurrency    FieldType = "currency"
    FieldPercent     FieldType = "percent"
    FieldToggle      FieldType = "toggle"
    FieldRadio       FieldType = "radio"
    FieldDynamicLink FieldType = "dynamic_link"
    FieldTime        FieldType = "time"
    FieldDuration    FieldType = "duration"
    FieldImage       FieldType = "image"
    FieldSignature   FieldType = "signature"
    FieldBarcode     FieldType = "barcode"
    FieldColor       FieldType = "color"
    FieldGeolocation FieldType = "geolocation"
    FieldRating      FieldType = "rating"
)
```

**Step 2: Add new fields to FieldDefinition struct**

```go
type FieldDefinition struct {
    // ... existing fields unchanged ...

    // Widget override
    Widget string `json:"widget,omitempty"`

    // Form behavior
    DependsOn   string `json:"depends_on,omitempty"`
    ReadOnlyIf  string `json:"readonly_if,omitempty"`
    MandatoryIf string `json:"mandatory_if,omitempty"`
    FetchFrom   string `json:"fetch_from,omitempty"`
    Formula     string `json:"formula,omitempty"`

    // Field-specific config
    Language  string `json:"language,omitempty"`
    Toolbar   string `json:"toolbar,omitempty"`
    CurrencyCode string `json:"currency,omitempty"`
    Format    string `json:"format,omitempty"`
    DrawMode  string `json:"draw_mode,omitempty"`
    MaxStars  int    `json:"max_stars,omitempty"`
    HalfStars bool   `json:"half_stars,omitempty"`
    Rows      int    `json:"rows,omitempty"`
    Accept    string `json:"accept,omitempty"`
}
```

**Step 3: Add validation for new field types in ParseModel**

Add validation cases for `FieldRadio` (needs options), `FieldDynamicLink` (needs model-field reference), etc.

**Step 4: Write tests for new field types**

```go
func TestParseModel_NewFieldTypes(t *testing.T) {
    json := `{
        "name": "test",
        "fields": {
            "amount":    {"type": "currency", "currency": "IDR", "precision": 0},
            "progress":  {"type": "percent", "min": 0, "max": 100},
            "code":      {"type": "code", "language": "python"},
            "rating":    {"type": "rating", "max_stars": 5, "half_stars": true},
            "color":     {"type": "color"},
            "location":  {"type": "geolocation", "draw_mode": "point"},
            "priority":  {"type": "radio", "options": ["low","medium","high"]},
            "active":    {"type": "toggle"},
            "start_time":{"type": "time"},
            "duration":  {"type": "duration"},
            "photo":     {"type": "image", "max_size": "5MB"},
            "sign":      {"type": "signature"},
            "barcode":   {"type": "barcode", "format": "qr"},
            "bio":       {"type": "richtext", "toolbar": "minimal"},
            "readme":    {"type": "markdown"},
            "snippet":   {"type": "smalltext", "rows": 3},
            "secret":    {"type": "password"},
            "price":     {"type": "float", "precision": 4},
            "content":   {"type": "html"},
            "ref":       {"type": "dynamic_link"},
            "name":      {"type": "string", "depends_on": "active == true", "readonly_if": "status != 'draft'"}
        }
    }`
    model, err := ParseModel([]byte(json))
    assert.NoError(t, err)
    assert.Equal(t, FieldCurrency, model.Fields["amount"].Type)
    assert.Equal(t, "IDR", model.Fields["amount"].CurrencyCode)
    assert.Equal(t, 5, model.Fields["rating"].MaxStars)
    assert.True(t, model.Fields["rating"].HalfStars)
    assert.Equal(t, "active == true", model.Fields["name"].DependsOn)
}
```

**Step 5: Run Go tests**

Run: `cd engine && go test ./internal/compiler/parser/ -v`
Expected: All tests pass including new ones

**Step 6: Commit**

```bash
git add engine/internal/compiler/parser/model.go engine/internal/compiler/parser/model_test.go
git commit -m "feat: add 17 new field types + widget/behavior fields to model parser"
```

---

### Task 5: Extend Go View Parser — New Layout Items

**Files:**
- Modify: `engine/internal/compiler/parser/view.go`
- Modify: `engine/internal/compiler/parser/view_test.go`

**Step 1: Add new structs to view.go**

```go
type HeaderDefinition struct {
    StatusField string             `json:"status_field,omitempty"`
    Widget      string             `json:"widget,omitempty"`
    Buttons     []ActionDefinition `json:"buttons,omitempty"`
}

type SmartButtonDefinition struct {
    Label       string  `json:"label"`
    Icon        string  `json:"icon,omitempty"`
    CountModel  string  `json:"count_model,omitempty"`
    CountDomain [][]any `json:"count_domain,omitempty"`
}

type SectionDefinition struct {
    Title       string `json:"title,omitempty"`
    Description string `json:"description,omitempty"`
    Collapsible bool   `json:"collapsible,omitempty"`
    CollapsedBy string `json:"collapsed_by,omitempty"`
}

type SeparatorDefinition struct {
    Label string `json:"label,omitempty"`
}

type ChildTableColumn struct {
    Field    string `json:"field"`
    Width    int    `json:"width,omitempty"`
    Readonly bool   `json:"readonly,omitempty"`
    Widget   string `json:"widget,omitempty"`
    Formula  string `json:"formula,omitempty"`
}

type ChildTableDefinition struct {
    Field   string            `json:"field"`
    Columns []ChildTableColumn `json:"columns"`
    Summary map[string]string `json:"summary,omitempty"`
}
```

**Step 2: Extend LayoutItem and LayoutRow**

```go
type LayoutRow struct {
    Field    string `json:"field,omitempty"`
    Width    int    `json:"width,omitempty"`
    Readonly bool   `json:"readonly,omitempty"`
    Widget   string `json:"widget,omitempty"`    // NEW
    Formula  string `json:"formula,omitempty"`   // NEW
}

type LayoutItem struct {
    // Existing
    Row  []LayoutRow     `json:"row,omitempty"`
    Tabs []TabDefinition `json:"tabs,omitempty"`

    // New
    Header     *HeaderDefinition       `json:"header,omitempty"`
    ButtonBox  []SmartButtonDefinition  `json:"button_box,omitempty"`
    Section    *SectionDefinition       `json:"section,omitempty"`
    Rows       []LayoutItem            `json:"rows,omitempty"`
    ChildTable *ChildTableDefinition   `json:"child_table,omitempty"`
    Chatter    bool                    `json:"chatter,omitempty"`
    Separator  *SeparatorDefinition    `json:"separator,omitempty"`
}
```

**Step 3: Write tests for extended layout**

```go
func TestParseView_ExtendedLayout(t *testing.T) {
    json := `{
        "name": "order_form",
        "type": "form",
        "model": "order",
        "layout": [
            {"header": {"status_field": "status", "widget": "statusbar", "buttons": [{"label": "Confirm", "process": "confirm"}]}},
            {"button_box": [{"label": "Invoices", "icon": "file-text", "count_model": "invoice"}]},
            {"section": {"title": "Info", "collapsible": true}, "rows": [
                {"row": [{"field": "name", "width": 6, "widget": "badge"}]}
            ]},
            {"child_table": {"field": "lines", "columns": [{"field": "product", "width": 6}], "summary": {"total": "sum"}}},
            {"chatter": true},
            {"separator": {"label": "Details"}}
        ]
    }`
    view, err := ParseView([]byte(json))
    assert.NoError(t, err)
    assert.NotNil(t, view.Layout[0].Header)
    assert.Equal(t, "statusbar", view.Layout[0].Header.Widget)
    assert.Len(t, view.Layout[1].ButtonBox, 1)
    assert.NotNil(t, view.Layout[2].Section)
    assert.True(t, view.Layout[2].Section.Collapsible)
    assert.NotNil(t, view.Layout[3].ChildTable)
    assert.True(t, view.Layout[4].Chatter)
    assert.NotNil(t, view.Layout[5].Separator)
}
```

**Step 4: Run Go tests**

Run: `cd engine && go test ./internal/compiler/parser/ -v`
Expected: All tests pass

**Step 5: Commit**

```bash
git add engine/internal/compiler/parser/view.go engine/internal/compiler/parser/view_test.go
git commit -m "feat: extend view parser with header, button_box, section, child_table, chatter, separator"
```

---

## Batch 2: Layout Components (Tasks 6-7)

### Task 6: Layout Components — lc-section, lc-row, lc-column, lc-separator, lc-sheet, lc-html-block

**Files:**
- Create: `packages/components/src/components/layout/lc-section/lc-section.tsx`
- Create: `packages/components/src/components/layout/lc-section/lc-section.css`
- Create: `packages/components/src/components/layout/lc-row/lc-row.tsx`
- Create: `packages/components/src/components/layout/lc-row/lc-row.css`
- Create: `packages/components/src/components/layout/lc-column/lc-column.tsx`
- Create: `packages/components/src/components/layout/lc-column/lc-column.css`
- Create: `packages/components/src/components/layout/lc-separator/lc-separator.tsx`
- Create: `packages/components/src/components/layout/lc-sheet/lc-sheet.tsx`
- Create: `packages/components/src/components/layout/lc-html-block/lc-html-block.tsx`

Each component follows the Stencil pattern: `@Component` decorator, `@Prop()` for attributes, `render()` method, Shadow DOM CSS.

**Step 1: Implement lc-section**

```tsx
// lc-section.tsx
import { Component, Prop, State, h } from '@stencil/core';

@Component({ tag: 'lc-section', styleUrl: 'lc-section.css', shadow: true })
export class LcSection {
  @Prop() sectionTitle: string;
  @Prop() description: string;
  @Prop() collapsible: boolean = false;
  @State() collapsed: boolean = false;

  render() {
    return (
      <section class={{ 'lc-section': true, 'collapsed': this.collapsed }}>
        {this.sectionTitle && (
          <div class="lc-section-header" onClick={() => this.collapsible && (this.collapsed = !this.collapsed)}>
            <h3>{this.sectionTitle}</h3>
            {this.description && <p class="description">{this.description}</p>}
            {this.collapsible && <span class="toggle">{this.collapsed ? '+' : '-'}</span>}
          </div>
        )}
        <div class="lc-section-body" style={{ display: this.collapsed ? 'none' : 'block' }}>
          <slot></slot>
        </div>
      </section>
    );
  }
}
```

**Step 2-6: Implement remaining layout components** (lc-row, lc-column, lc-separator, lc-sheet, lc-html-block)

Each follows the same pattern. lc-row uses flexbox, lc-column uses CSS grid with width prop (1-12), etc.

**Step 7: Build and verify**

Run: `npm run build`
Expected: Build succeeds

**Step 8: Commit**

```bash
git add packages/components/src/components/layout/
git commit -m "feat: add layout components — section, row, column, separator, sheet, html-block"
```

---

### Task 7: Layout Components — lc-tabs, lc-tab, lc-header, lc-button-box

**Files:**
- Create: `packages/components/src/components/layout/lc-tabs/lc-tabs.tsx`
- Create: `packages/components/src/components/layout/lc-tab/lc-tab.tsx`
- Create: `packages/components/src/components/layout/lc-header/lc-header.tsx`
- Create: `packages/components/src/components/layout/lc-button-box/lc-button-box.tsx`

**Step 1-4: Implement each component**

lc-tabs manages active tab state, lc-tab is a pane. lc-header renders status bar + action buttons. lc-button-box renders smart buttons with counts.

**Step 5: Commit**

```bash
git add packages/components/src/components/layout/
git commit -m "feat: add tabs, tab, header, button-box layout components"
```

---

## Batch 3: Field Components (Tasks 8-14)

### Task 8: Basic Field Components — string, smalltext, text, integer, float, password

6 simple input components. Each emits `lc-field-change` CustomEvent on value change.

### Task 9: Number Field Components — currency, percent, decimal

3 number components with formatting.

### Task 10: Selection Field Components — select, checkbox, toggle, radio

4 selection components.

### Task 11: Date/Time Field Components — date, time, datetime, duration

4 date/time components with pickers.

### Task 12: Rich Content Field Components — richtext, markdown, code, html, json

5 editor components. richtext uses TipTap, code/json use CodeMirror 6, markdown uses CodeMirror + markdown-it.

### Task 13: Relation Field Components — link, dynlink, tags, multicheck, tableselect

5 relation components. link has autocomplete search + quick-create dialog.

### Task 14: Special Field Components — file, image, signature, barcode, color, geo, rating

7 special components. geo uses Leaflet, barcode uses JsBarcode, signature uses signature_pad.

---

## Batch 4: Widget Components (Tasks 15-16)

### Task 15: Widget Components — statusbar, badge, priority, handle, copy

5 widget components.

### Task 16: Widget Components — phone, email, url, progress, domain

5 widget components.

---

## Batch 5: View Components (Tasks 17-22)

### Task 17: List View Component — lc-view-list

Full list view with sorting, filtering, pagination, bulk select, row decoration, column optional.

### Task 18: Form View Component — lc-view-form

Form view that reads layout definition and renders field/layout components. Integrates with FormEngine.

### Task 19: Kanban View Component — lc-view-kanban

Kanban board with drag-drop (SortableJS), quick create, card template, progress bar per column.

### Task 20: Calendar & Gantt View Components

Calendar uses FullCalendar. Gantt uses frappe-gantt or custom.

### Task 21: Tree, Map, Activity View Components

Tree for hierarchical data. Map uses Leaflet. Activity shows timeline.

### Task 22: Report Builder Component — lc-view-report

Query report with custom columns, filters, totals, chart integration.

---

## Batch 6: Chart & Dashboard Components (Tasks 23-24)

### Task 23: Chart Components — kpi, bar, line, pie, area, gauge, progress

7 chart components using ECharts.

### Task 24: Chart Components — heatmap, funnel, pivot, scorecard

4 advanced chart components.

---

## Batch 7: Complex Components (Tasks 25-28)

### Task 25: Child Table Component — lc-child-table

Inline editable table with add/delete row, drag reorder, any field type per column, subtotals.

### Task 26: Dialog Components — modal, wizard, quickentry, confirm, toast

5 dialog/overlay components.

### Task 27: Chatter & Communication — chatter, activity, timeline

3 social components.

### Task 28: Search & Filter — search, filter-panel, filter-bar, favorites

4 search components.

---

## Batch 8: Print, Export & Integration (Tasks 29-32)

### Task 29: Print & Export Components — print, export, report-link

3 print/export components.

### Task 30: Go Engine — Rewrite ViewRenderer to Output Custom Elements

**Files:**
- Modify: `engine/internal/presentation/view/renderer.go`
- Create: `engine/internal/presentation/view/component_compiler.go`
- Create: `engine/internal/presentation/view/component_compiler_test.go`

Rewrite all render methods (renderForm, renderList, renderKanban, etc.) to output `<lc-*>` custom elements instead of raw HTML.

### Task 31: Go Engine — Static Asset Handler for Stencil Bundle

**Files:**
- Create: `engine/internal/presentation/assets/handler.go`
- Modify: `engine/internal/app.go` (wire asset handler)

Serve Stencil dist/ at `/assets/components/`. Add `<script>` tag to layout template.

### Task 32: Integration — Update Sample ERP to Use New Components

**Files:**
- Modify: `samples/erp/modules/crm/views/*.json` (use new layout features)
- Modify: `samples/erp/modules/hrm/views/*.json`
- Modify: `engine/modules/sales/views/*.json`
- Modify: `samples/erp/modules/crm/models/*.json` (add widget, depends_on, etc.)

Update all existing view and model JSON files to use the new extended schema.

---

## Batch 9: Testing & Polish (Tasks 33-35)

### Task 33: Stencil Unit Tests for All Components

Run: `npm test`
Expected: All component spec tests pass

### Task 34: Go Integration Tests

Run: `cd engine && go test ./... -v`
Expected: All 93+ tests pass, including new parser tests

### Task 35: End-to-End Verification

Start engine with sample ERP, verify all views render with new components, test form behavior, kanban drag-drop, child table, chatter.

---

## Summary

| Batch | Tasks | Components | Effort |
|---|---|---|---|
| 1. Foundation | 1-5 | Stencil scaffold, core utils, form engine, Go parser extensions | High |
| 2. Layout | 6-7 | 10 layout components | Medium |
| 3. Fields | 8-14 | 30 field components | High |
| 4. Widgets | 15-16 | 10 widget components | Medium |
| 5. Views | 17-22 | 9 view components | High |
| 6. Charts | 23-24 | 11 chart components | Medium |
| 7. Complex | 25-28 | 13 complex components (table, dialogs, social, search) | High |
| 8. Integration | 29-32 | 3 print + Go engine rewrite + sample update | High |
| 9. Testing | 33-35 | Tests and verification | Medium |

**Total: 35 tasks, 86 components, ~9 batches**
