# Low-Code ERP Component System — Design Document

**Date:** 2026-04-18
**Status:** Approved
**Sprint:** 1.10 (fix-1-2)
**Approach:** Stencil Monorepo + Go Engine JSON-to-HTML Compiler

---

## 1. Overview

This design adds a comprehensive component system to the low-code ERP platform. The system provides **86 ready-to-use components** covering form fields, layout, views, charts, widgets, dialogs, and more — inspired by the best of ERPNext (Frappe) and Odoo.

**Key requirement:** Every component works in two modes:
1. **JSON path (low-code):** User writes `view.json` -> Go engine compiles -> HTML containing `<lc-*>` custom elements -> browser hydrates
2. **HTML path (pro-code):** User writes HTML directly using `<lc-field-currency value="1000">` in custom templates

**Technology choice:** Stencil (by Ionic) as the Web Components compiler. Stencil provides compile-time optimization, lazy loading per component, and auto-generates framework wrappers (React, Vue, Angular).

---

## 2. Architecture

```
+--------------------------------------------------+
|              User's Module (JSON)                 |
|  model.json -> api.json -> view.json              |
+------------------------+-------------------------+
                         | parsed by
+------------------------v-------------------------+
|              Go Engine (existing)                 |
|  Parser -> ModelRegistry -> ViewRenderer          |
|                                                   |
|  NEW: JSON-to-HTML compiler outputs <lc-*>        |
|       custom elements                             |
+------------------------+-------------------------+
                         | serves HTML containing
+------------------------v-------------------------+
|         Stencil Component Library (NEW)           |
|  packages/components/                             |
|  +-- fields/     30 field components              |
|  +-- layout/     10 layout components             |
|  +-- views/       9 view components               |
|  +-- charts/     11 chart/dashboard components    |
|  +-- widgets/    10 widget components             |
|  +-- dialogs/     5 dialog/overlay components     |
|  +-- table/       1 child table (complex)         |
|  +-- social/      3 chatter/communication         |
|  +-- search/      4 search/filter components      |
|  +-- print/       3 print/export components       |
|  +-- core/        form behavior engine            |
|                                                   |
|  Output: lazy-loaded JS bundle + wrappers         |
+---------------------------------------------------+
```

### Data Flow

```
1. User writes view.json
2. Go engine parses -> ViewDefinition (extended struct)
3. Go ViewRenderer compiles -> HTML string containing <lc-*> elements
4. HTML served to browser with <script src="/assets/lc-components.js">
5. Stencil runtime hydrates custom elements -> interactive UI
6. User interactions -> CustomEvents -> API calls back to Go engine
```

---

## 3. Complete Component Catalog (86 components)

### 3.1 Field Components (30)

#### Text & Number (12)

| Component | Tag | JSON type | Key Props |
|---|---|---|---|
| Data/Char | `<lc-field-string>` | `string` | value, max, placeholder, validate(email/url/phone) |
| Small Text | `<lc-field-smalltext>` | `smalltext` | value, rows(3) |
| Long Text | `<lc-field-text>` | `text` | value, maxlength |
| Rich Text Editor | `<lc-field-richtext>` | `richtext` | value, toolbar(full/minimal) |
| Markdown Editor | `<lc-field-markdown>` | `markdown` | value, preview(true/false) |
| HTML Display | `<lc-field-html>` | `html` | content |
| Code Editor | `<lc-field-code>` | `code` | value, language(python/js/sql/xml) |
| Password | `<lc-field-password>` | `password` | value, strength-meter(true/false) |
| Integer | `<lc-field-integer>` | `integer` | value, min, max, step |
| Float | `<lc-field-float>` | `float` | value, precision(9) |
| Currency | `<lc-field-currency>` | `currency` | value, currency(IDR/USD), precision(2) |
| Percent | `<lc-field-percent>` | `percent` | value, min(0), max(100) |

#### Selection & Relations (10)

| Component | Tag | JSON type | Key Props |
|---|---|---|---|
| Select/Dropdown | `<lc-field-select>` | `selection` | value, options[], placeholder |
| Checkbox | `<lc-field-checkbox>` | `boolean` | value, label |
| Toggle | `<lc-field-toggle>` | `toggle` | value, label |
| Radio Group | `<lc-field-radio>` | `radio` | value, options[], direction(h/v) |
| Link (Many2one) | `<lc-field-link>` | `many2one` | value, model, display-field, quick-create |
| Dynamic Link | `<lc-field-dynlink>` | `dynamic_link` | value, model-field |
| Tags (Many2many) | `<lc-field-tags>` | `many2many` | values[], model, color(true/false) |
| Multi-Checkbox | `<lc-field-multicheck>` | `many2many_check` | values[], options[] |
| Table MultiSelect | `<lc-field-tableselect>` | `table_multiselect` | values[], model |
| Multi-Checkbox List | `<lc-field-checklist>` | `many2many_check` | values[], options[] |

#### Date & Time (4)

| Component | Tag | JSON type | Key Props |
|---|---|---|---|
| Date | `<lc-field-date>` | `date` | value, min, max, format |
| Time | `<lc-field-time>` | `time` | value, format(12h/24h) |
| DateTime | `<lc-field-datetime>` | `datetime` | value |
| Duration | `<lc-field-duration>` | `duration` | value, units(days/hours/minutes/seconds) |

#### File & Media (3)

| Component | Tag | JSON type | Key Props |
|---|---|---|---|
| File Upload | `<lc-field-file>` | `file` | value, accept, max-size |
| Image Upload | `<lc-field-image>` | `image` | value, accept, max-size, preview |
| Signature | `<lc-field-signature>` | `signature` | value, width, height |

#### Special & Advanced (5)

| Component | Tag | JSON type | Key Props |
|---|---|---|---|
| Barcode | `<lc-field-barcode>` | `barcode` | value, format(code128/qr/ean13) |
| Color Picker | `<lc-field-color>` | `color` | value, format(hex/rgb) |
| Geolocation | `<lc-field-geo>` | `geolocation` | value(lat,lng), draw(point/line/polygon) |
| Rating | `<lc-field-rating>` | `rating` | value, max(5), half(true/false) |
| JSON Editor | `<lc-field-json>` | `json` | value, schema |

### 3.2 Layout Components (10)

| Component | Tag | Function |
|---|---|---|
| Section | `<lc-section>` | Horizontal section with optional title, description, collapsible |
| Column | `<lc-column>` | Column within section (auto-grid, width 1-12) |
| Row | `<lc-row>` | Flex row container |
| Tabs | `<lc-tabs>` | Tab navigation container |
| Tab | `<lc-tab>` | Single tab pane |
| Header Bar | `<lc-header>` | Form header with workflow buttons + status bar |
| Button Box | `<lc-button-box>` | Smart button area (stat buttons with counts) |
| Separator | `<lc-separator>` | Visual separator with optional label |
| HTML Block | `<lc-html-block>` | Static HTML content insertion |
| Sheet | `<lc-sheet>` | Responsive form wrapper |

### 3.3 View Components (9)

| Component | Tag | Function |
|---|---|---|
| List View | `<lc-view-list>` | Tabular data with sort, filter, group, pagination, bulk actions |
| Form View | `<lc-view-form>` | Detail/edit form with layout engine |
| Kanban View | `<lc-view-kanban>` | Drag-drop board grouped by field |
| Calendar View | `<lc-view-calendar>` | Calendar with date-based records |
| Gantt View | `<lc-view-gantt>` | Timeline/project management |
| Map View | `<lc-view-map>` | Geolocation records on Leaflet map |
| Tree View | `<lc-view-tree>` | Hierarchical parent-child (chart of accounts, BOM) |
| Activity View | `<lc-view-activity>` | Timeline log of activities and communications |
| Report Builder | `<lc-view-report>` | Query report with custom columns, filters, totals |

### 3.4 Chart & Dashboard Components (11)

| Component | Tag | Function |
|---|---|---|
| KPI Card | `<lc-chart-kpi>` | Big number + label + trend indicator (up/down) |
| Bar Chart | `<lc-chart-bar>` | Vertical/horizontal bars (comparison) |
| Line Chart | `<lc-chart-line>` | Time series trends |
| Pie/Donut | `<lc-chart-pie>` | Proportional distribution |
| Area Chart | `<lc-chart-area>` | Filled line chart |
| Heatmap | `<lc-chart-heatmap>` | Color intensity grid (activity per day/month) |
| Funnel | `<lc-chart-funnel>` | Pipeline/conversion funnel |
| Gauge | `<lc-chart-gauge>` | Percentage circle/gauge |
| Progress Bar | `<lc-chart-progress>` | Horizontal progress bar |
| Pivot Table | `<lc-chart-pivot>` | Cross-tab aggregation |
| Scorecard | `<lc-chart-scorecard>` | Metric vs target comparison |

### 3.5 Widget Components (10)

Widgets change how a field is rendered without changing its data type.

| Component | Tag | Function |
|---|---|---|
| Status Bar | `<lc-widget-statusbar>` | Arrow/segmented workflow visualization |
| Priority Stars | `<lc-widget-priority>` | Star-based priority indicator |
| Handle | `<lc-widget-handle>` | Drag handle for reorder in lists |
| Badge | `<lc-widget-badge>` | Colored pill read-only display |
| Copy Button | `<lc-widget-copy>` | Copy value to clipboard |
| Phone Link | `<lc-widget-phone>` | Clickable/dialable phone number |
| Email Link | `<lc-widget-email>` | Clickable mailto link |
| URL Link | `<lc-widget-url>` | Clickable URL with external icon |
| Progress Column | `<lc-widget-progress>` | Progress bar in list view column |
| Domain Builder | `<lc-widget-domain>` | Visual filter/condition builder |

### 3.6 Dialog & Overlay Components (5)

| Component | Tag | Function |
|---|---|---|
| Modal | `<lc-dialog-modal>` | Generic modal dialog |
| Wizard | `<lc-dialog-wizard>` | Multi-step dialog (next/back/finish) |
| Quick Entry | `<lc-dialog-quickentry>` | Mini form for creating linked records inline |
| Confirm | `<lc-dialog-confirm>` | Confirmation before destructive actions |
| Toast | `<lc-toast>` | Corner notification (success/error/warning/info) |

### 3.7 Child Table (1, complex)

| Component | Tag | Function |
|---|---|---|
| Child Table | `<lc-child-table>` | Inline editable table: add/delete row, drag reorder, any field type per column, subtotals, summary row |

### 3.8 Chatter & Communication (3)

| Component | Tag | Function |
|---|---|---|
| Chatter | `<lc-chatter>` | Message thread + activity log + followers |
| Activity Widget | `<lc-activity>` | Schedule & track activities (call, meeting, email) |
| Timeline | `<lc-timeline>` | Audit trail of field changes (who changed what, when) |

### 3.9 Search & Filter (4)

| Component | Tag | Function |
|---|---|---|
| Search Box | `<lc-search>` | Main search with dropdown suggestions |
| Filter Panel | `<lc-filter-panel>` | Side panel with category filters + counts |
| Filter Bar | `<lc-filter-bar>` | Preset filter buttons (My Records, Overdue, etc.) |
| Favorites | `<lc-favorites>` | Save/load filter + group-by combinations |

### 3.10 Print & Export (3)

| Component | Tag | Function |
|---|---|---|
| Print Format | `<lc-print>` | PDF/HTML print template with letterhead |
| Export Button | `<lc-export>` | Export to XLSX/CSV |
| Report Shortcut | `<lc-report-link>` | Dashboard link to report/view |

---

## 4. JSON Schema Extensions

### 4.1 New Field Types (17 additions to existing 15)

```
Existing: string, text, integer, decimal, boolean, date, datetime,
          selection, email, many2one, one2many, many2many, json, file, computed

New:      smalltext, richtext, markdown, html, code, password, float,
          currency, percent, toggle, radio, dynamic_link, time, duration,
          image, signature, barcode, color, geolocation, rating
```

Total: 32 field types.

### 4.2 Extended FieldDefinition (Go struct)

```go
type FieldDefinition struct {
    // --- Existing fields (unchanged) ---
    Type      FieldType `json:"type"`
    Label     string    `json:"label,omitempty"`
    Required  bool      `json:"required,omitempty"`
    Unique    bool      `json:"unique,omitempty"`
    Default   any       `json:"default,omitempty"`
    Max       int       `json:"max,omitempty"`
    Min       int       `json:"min,omitempty"`
    Precision int       `json:"precision,omitempty"`
    MaxSize   string    `json:"max_size,omitempty"`
    Options   []string  `json:"options,omitempty"`
    Model     string    `json:"model,omitempty"`
    Inverse   string    `json:"inverse,omitempty"`
    Computed  string    `json:"computed,omitempty"`
    Auto      bool      `json:"auto,omitempty"`

    // --- NEW: Widget override ---
    Widget string `json:"widget,omitempty"` // "statusbar", "badge", "priority", etc.

    // --- NEW: Form behavior ---
    DependsOn   string `json:"depends_on,omitempty"`   // "status == 'draft'"
    ReadOnlyIf  string `json:"readonly_if,omitempty"`  // "status != 'draft'"
    MandatoryIf string `json:"mandatory_if,omitempty"` // "type == 'company'"
    FetchFrom   string `json:"fetch_from,omitempty"`   // "contact_id.email"
    Formula     string `json:"formula,omitempty"`       // "qty * unit_price"

    // --- NEW: Field-specific config ---
    Language  string `json:"language,omitempty"`   // code editor language
    Toolbar   string `json:"toolbar,omitempty"`    // richtext toolbar mode
    Currency  string `json:"currency,omitempty"`   // currency symbol/code
    Format    string `json:"format,omitempty"`     // barcode format, date format
    DrawMode  string `json:"draw_mode,omitempty"`  // geolocation draw mode
    MaxStars  int    `json:"max_stars,omitempty"`  // rating max stars
    HalfStars bool   `json:"half_stars,omitempty"` // rating half-star support
    Rows      int    `json:"rows,omitempty"`       // smalltext visible rows
    Accept    string `json:"accept,omitempty"`     // file/image accepted types
}
```

### 4.3 Extended View Layout Schema

The view layout JSON gains new layout item types:

```json
{
  "name": "order_form",
  "type": "form",
  "model": "order",
  "layout": [
    {
      "header": {
        "status_field": "status",
        "widget": "statusbar",
        "buttons": [
          { "label": "Confirm", "process": "confirm_order", "variant": "primary", "visible": "status == 'draft'" }
        ]
      }
    },
    {
      "button_box": [
        { "label": "Invoices", "icon": "file-text", "count_model": "invoice", "count_domain": [["order_id", "=", "{{id}}"]] },
        { "label": "Deliveries", "icon": "truck", "count_model": "delivery", "count_domain": [["order_id", "=", "{{id}}"]] }
      ]
    },
    {
      "section": { "title": "Order Info", "collapsible": false },
      "rows": [
        { "row": [
          { "field": "customer_id", "width": 6 },
          { "field": "order_date", "width": 3 },
          { "field": "status", "width": 3, "widget": "badge" }
        ]}
      ]
    },
    {
      "section": { "title": "Order Lines", "collapsible": true },
      "rows": [
        { "child_table": {
          "field": "lines",
          "columns": [
            { "field": "product_id", "width": 4 },
            { "field": "qty", "width": 2 },
            { "field": "unit_price", "width": 2, "widget": "currency" },
            { "field": "subtotal", "width": 2, "readonly": true, "formula": "qty * unit_price" }
          ],
          "summary": { "subtotal": "sum" }
        }}
      ]
    },
    {
      "tabs": [
        { "label": "Notes", "fields": ["notes"] },
        { "label": "Terms", "fields": ["payment_terms", "delivery_terms"] }
      ]
    },
    { "chatter": true }
  ]
}
```

New layout item types:
- `header` — Form header with status bar and action buttons
- `button_box` — Smart buttons with related record counts
- `section` — Named section with collapsible support
- `child_table` — Inline editable table with column definitions and summary
- `chatter` — Communication panel (boolean flag)
- `separator` — Visual divider

### 4.4 Extended Go View Parser Structs

```go
type HeaderDefinition struct {
    StatusField string             `json:"status_field,omitempty"`
    Widget      string             `json:"widget,omitempty"`
    Buttons     []ActionDefinition `json:"buttons,omitempty"`
}

type SmartButtonDefinition struct {
    Label      string  `json:"label"`
    Icon       string  `json:"icon,omitempty"`
    CountModel string  `json:"count_model,omitempty"`
    CountDomain [][]any `json:"count_domain,omitempty"`
}

type SectionDefinition struct {
    Title       string `json:"title,omitempty"`
    Description string `json:"description,omitempty"`
    Collapsible bool   `json:"collapsible,omitempty"`
    CollapsedBy string `json:"collapsed_by,omitempty"` // expression
}

type ChildTableColumn struct {
    Field    string `json:"field"`
    Width    int    `json:"width,omitempty"`
    Readonly bool   `json:"readonly,omitempty"`
    Widget   string `json:"widget,omitempty"`
    Formula  string `json:"formula,omitempty"`
}

type ChildTableDefinition struct {
    Field   string                    `json:"field"`
    Columns []ChildTableColumn        `json:"columns"`
    Summary map[string]string         `json:"summary,omitempty"` // field -> "sum"/"avg"/"count"
}

type LayoutItem struct {
    // Existing
    Row  []LayoutRow     `json:"row,omitempty"`
    Tabs []TabDefinition `json:"tabs,omitempty"`

    // New
    Header     *HeaderDefinition       `json:"header,omitempty"`
    ButtonBox  []SmartButtonDefinition  `json:"button_box,omitempty"`
    Section    *SectionDefinition       `json:"section,omitempty"`
    Rows       []LayoutItem            `json:"rows,omitempty"`       // rows within a section
    ChildTable *ChildTableDefinition   `json:"child_table,omitempty"`
    Chatter    bool                    `json:"chatter,omitempty"`
    Separator  *SeparatorDefinition    `json:"separator,omitempty"`
}
```

---

## 5. Stencil Project Structure

```
packages/components/
+-- stencil.config.ts
+-- package.json
+-- tsconfig.json
+-- src/
|   +-- components/
|   |   +-- fields/
|   |   |   +-- lc-field-string/
|   |   |   |   +-- lc-field-string.tsx
|   |   |   |   +-- lc-field-string.css
|   |   |   |   +-- lc-field-string.spec.ts
|   |   |   +-- lc-field-smalltext/
|   |   |   +-- lc-field-text/
|   |   |   +-- lc-field-richtext/
|   |   |   +-- lc-field-markdown/
|   |   |   +-- lc-field-html/
|   |   |   +-- lc-field-code/
|   |   |   +-- lc-field-password/
|   |   |   +-- lc-field-integer/
|   |   |   +-- lc-field-float/
|   |   |   +-- lc-field-currency/
|   |   |   +-- lc-field-percent/
|   |   |   +-- lc-field-select/
|   |   |   +-- lc-field-checkbox/
|   |   |   +-- lc-field-toggle/
|   |   |   +-- lc-field-radio/
|   |   |   +-- lc-field-link/
|   |   |   +-- lc-field-dynlink/
|   |   |   +-- lc-field-tags/
|   |   |   +-- lc-field-multicheck/
|   |   |   +-- lc-field-tableselect/
|   |   |   +-- lc-field-date/
|   |   |   +-- lc-field-time/
|   |   |   +-- lc-field-datetime/
|   |   |   +-- lc-field-duration/
|   |   |   +-- lc-field-file/
|   |   |   +-- lc-field-image/
|   |   |   +-- lc-field-signature/
|   |   |   +-- lc-field-barcode/
|   |   |   +-- lc-field-color/
|   |   |   +-- lc-field-geo/
|   |   |   +-- lc-field-rating/
|   |   |   +-- lc-field-json/
|   |   +-- layout/
|   |   |   +-- lc-section/
|   |   |   +-- lc-column/
|   |   |   +-- lc-row/
|   |   |   +-- lc-tabs/
|   |   |   +-- lc-tab/
|   |   |   +-- lc-header/
|   |   |   +-- lc-button-box/
|   |   |   +-- lc-separator/
|   |   |   +-- lc-html-block/
|   |   |   +-- lc-sheet/
|   |   +-- views/
|   |   |   +-- lc-view-list/
|   |   |   +-- lc-view-form/
|   |   |   +-- lc-view-kanban/
|   |   |   +-- lc-view-calendar/
|   |   |   +-- lc-view-gantt/
|   |   |   +-- lc-view-map/
|   |   |   +-- lc-view-tree/
|   |   |   +-- lc-view-activity/
|   |   |   +-- lc-view-report/
|   |   +-- charts/
|   |   |   +-- lc-chart-kpi/
|   |   |   +-- lc-chart-bar/
|   |   |   +-- lc-chart-line/
|   |   |   +-- lc-chart-pie/
|   |   |   +-- lc-chart-area/
|   |   |   +-- lc-chart-heatmap/
|   |   |   +-- lc-chart-funnel/
|   |   |   +-- lc-chart-gauge/
|   |   |   +-- lc-chart-progress/
|   |   |   +-- lc-chart-pivot/
|   |   |   +-- lc-chart-scorecard/
|   |   +-- widgets/
|   |   |   +-- lc-widget-statusbar/
|   |   |   +-- lc-widget-priority/
|   |   |   +-- lc-widget-handle/
|   |   |   +-- lc-widget-badge/
|   |   |   +-- lc-widget-copy/
|   |   |   +-- lc-widget-phone/
|   |   |   +-- lc-widget-email/
|   |   |   +-- lc-widget-url/
|   |   |   +-- lc-widget-progress/
|   |   |   +-- lc-widget-domain/
|   |   +-- dialogs/
|   |   |   +-- lc-dialog-modal/
|   |   |   +-- lc-dialog-wizard/
|   |   |   +-- lc-dialog-quickentry/
|   |   |   +-- lc-dialog-confirm/
|   |   |   +-- lc-toast/
|   |   +-- table/
|   |   |   +-- lc-child-table/
|   |   +-- social/
|   |   |   +-- lc-chatter/
|   |   |   +-- lc-activity/
|   |   |   +-- lc-timeline/
|   |   +-- search/
|   |   |   +-- lc-search/
|   |   |   +-- lc-filter-panel/
|   |   |   +-- lc-filter-bar/
|   |   |   +-- lc-favorites/
|   |   +-- print/
|   |       +-- lc-print/
|   |       +-- lc-export/
|   |       +-- lc-report-link/
|   +-- core/
|   |   +-- form-engine.ts          # Form behavior (depends_on, readonly_if, etc.)
|   |   +-- api-client.ts           # HTTP client for Go engine APIs
|   |   +-- event-bus.ts            # Client-side event bus
|   |   +-- i18n.ts                 # Client-side translations
|   |   +-- theme.ts                # CSS custom properties / theming
|   |   +-- types.ts                # Shared TypeScript types
|   |   +-- websocket.ts            # WebSocket client for real-time updates
|   +-- utils/
|   |   +-- expression-eval.ts      # Safe expression evaluator for depends_on etc.
|   |   +-- format.ts               # Number, date, currency formatters
|   |   +-- validators.ts           # Client-side validation
|   +-- global/
|       +-- global.css              # Design tokens, CSS custom properties
+-- dist/                           # Build output -> copied to engine/static/
+-- framework-wrappers/             # Auto-generated by Stencil
    +-- react/
    +-- vue/
    +-- angular/
```

---

## 6. Form Behavior Engine (Client-Side)

The `core/form-engine.ts` handles reactive form logic. It listens to `lc-field-change` events from any field component, re-evaluates all expressions, and updates affected fields.

### Supported behaviors:

| Behavior | JSON key | Example | Effect |
|---|---|---|---|
| Conditional visibility | `depends_on` | `"status == 'draft'"` | Show/hide field |
| Conditional readonly | `readonly_if` | `"status != 'draft'"` | Make field read-only |
| Conditional mandatory | `mandatory_if` | `"type == 'company'"` | Make field required |
| Auto-populate | `fetch_from` | `"contact_id.email"` | Fill from linked record |
| Formula | `formula` | `"qty * unit_price"` | Auto-calculate value |
| Default value | `default` | `"now"` / `"draft"` / `0` | Initial value on create |
| Naming series | `naming_series` | `"SINV-{YYYY}-{####}"` | Auto-generate document number |

### Expression evaluator:

Safe sandboxed evaluator (no eval/Function). Supports:
- Comparison: `==`, `!=`, `>`, `<`, `>=`, `<=`
- Logic: `&&`, `||`, `!`
- Arithmetic: `+`, `-`, `*`, `/`
- Field references: `field_name`, `linked_field.sub_field`
- Functions: `sum()`, `count()`, `avg()`, `min()`, `max()`

---

## 7. Go Engine Changes

### Files to modify:

| File | Change |
|---|---|
| `internal/compiler/parser/model.go` | Add 17 new FieldType constants + new FieldDefinition fields |
| `internal/compiler/parser/model_test.go` | Tests for new field types |
| `internal/compiler/parser/view.go` | Add Header, ButtonBox, Section, ChildTable, Chatter structs to LayoutItem |
| `internal/compiler/parser/view_test.go` | Tests for extended layout |
| `internal/presentation/view/renderer.go` | Rewrite to output `<lc-*>` custom elements |

### Files to add:

| File | Purpose |
|---|---|
| `internal/presentation/view/component_compiler.go` | JSON layout -> HTML with custom elements |
| `internal/presentation/view/component_compiler_test.go` | Tests |
| `internal/presentation/assets/handler.go` | Serve Stencil dist/ as static assets at `/assets/` |

### Renderer rewrite:

The current renderer generates raw HTML strings. The new renderer generates HTML containing `<lc-*>` custom elements:

```go
// Before: raw HTML
func (r *Renderer) renderForm(...) {
    return `<div class="lc-card"><form>...</form></div>`
}

// After: custom elements
func (r *Renderer) renderForm(...) {
    var html strings.Builder
    html.WriteString(`<lc-view-form model="` + viewDef.Model + `">`)
    for _, item := range viewDef.Layout {
        html.WriteString(r.compileLayoutItem(item))
    }
    html.WriteString(`</lc-view-form>`)
    return html.String()
}
```

---

## 8. Theming & Design System

CSS custom properties for consistent theming across all components:

```css
:root {
  /* Colors */
  --lc-primary: #4f46e5;
  --lc-success: #10b981;
  --lc-warning: #f59e0b;
  --lc-danger: #ef4444;
  --lc-info: #3b82f6;

  /* Typography */
  --lc-font-family: 'Inter', system-ui, sans-serif;
  --lc-font-size-sm: 0.875rem;
  --lc-font-size-base: 1rem;

  /* Spacing */
  --lc-spacing-xs: 0.25rem;
  --lc-spacing-sm: 0.5rem;
  --lc-spacing-md: 1rem;
  --lc-spacing-lg: 1.5rem;

  /* Borders */
  --lc-border-radius: 0.375rem;
  --lc-border-color: #e5e7eb;

  /* Shadows */
  --lc-shadow-sm: 0 1px 2px rgba(0,0,0,0.05);
  --lc-shadow-md: 0 4px 6px rgba(0,0,0,0.1);
}
```

All components use these tokens. Users can override them for custom themes.

---

## 9. Third-Party Dependencies (Stencil Components)

| Component | Library | Why |
|---|---|---|
| Rich Text Editor | TipTap (ProseMirror) | Headless, extensible, framework-agnostic |
| Code Editor | CodeMirror 6 | Lightweight, modular, web-component friendly |
| Markdown Editor | CodeMirror 6 + markdown-it | Preview rendering |
| Charts | Apache ECharts | Comprehensive, performant, SSR-capable |
| Calendar View | FullCalendar | Industry standard, plugin architecture |
| Gantt View | frappe-gantt or custom | Lightweight gantt |
| Map/Geolocation | Leaflet | Lightweight, open-source |
| Date Picker | Native + Flatpickr fallback | Lightweight |
| Drag & Drop | SortableJS | Framework-agnostic, touch support |
| Barcode | JsBarcode + qrcode | Generation + QR |
| Signature | signature_pad | Canvas-based |
| PDF Viewer | pdf.js | Mozilla's PDF renderer |
| Pivot Table | PivotTable.js or custom | Aggregation |

---

## 10. Event System (Component Communication)

All components communicate via CustomEvents on the DOM:

| Event | Emitted by | Payload |
|---|---|---|
| `lc-field-change` | Any field | `{ name, value, oldValue }` |
| `lc-field-focus` | Any field | `{ name }` |
| `lc-field-blur` | Any field | `{ name, value }` |
| `lc-form-submit` | Form view | `{ model, data }` |
| `lc-form-save` | Form view | `{ model, data, id }` |
| `lc-action-click` | Action button | `{ process, recordId }` |
| `lc-row-select` | List/table | `{ ids[] }` |
| `lc-kanban-move` | Kanban view | `{ id, from, to }` |
| `lc-dialog-open` | Dialog | `{ type }` |
| `lc-dialog-close` | Dialog | `{ type, result }` |
| `lc-toast-show` | Toast | `{ type, message }` |

The form-engine listens to `lc-field-change` and triggers re-evaluation of all expressions.

---

## 11. API Client (Component -> Engine)

The `core/api-client.ts` provides a typed HTTP client for components to communicate with the Go engine:

```typescript
class LcApiClient {
  // CRUD
  list(model: string, params?: ListParams): Promise<ListResponse>
  read(model: string, id: string): Promise<Record>
  create(model: string, data: Record): Promise<Record>
  update(model: string, id: string, data: Record): Promise<Record>
  delete(model: string, id: string): Promise<void>

  // Workflow
  action(model: string, id: string, action: string): Promise<Record>

  // Search
  search(model: string, query: string): Promise<Record[]>

  // File
  upload(file: File): Promise<{ url: string }>

  // WebSocket
  subscribe(channel: string, callback: (data: any) => void): void
}
```

Components access this via a global `window.__lc_api` or Stencil's dependency injection.

---

## 12. Build & Integration Pipeline

```
1. Stencil build (packages/components/)
   npm run build
   -> dist/lc-components/        (lazy-loaded bundles)
   -> dist/lc-components.js      (loader script)

2. Copy to Go engine static assets
   cp -r dist/* engine/static/components/

3. Go engine serves at /assets/components/
   <script type="module" src="/assets/components/lc-components.js"></script>

4. Go templates include the script tag in layout.html
   All <lc-*> elements auto-hydrate on page load
```

For development:
- Stencil dev server with hot reload for component development
- Go engine dev mode serves from packages/components/dist/ directly

---

## 13. Success Criteria

- [ ] All 86 components render correctly in both JSON and HTML modes
- [ ] Form behavior engine evaluates depends_on, readonly_if, mandatory_if, fetch_from, formula
- [ ] Child table supports add/delete/reorder rows with any field type
- [ ] All 9 view types render with real data from Go engine
- [ ] Charts render with ECharts using data from API
- [ ] Kanban drag-drop updates record via API
- [ ] Chatter shows message thread and activity log
- [ ] Search/filter components work with list and kanban views
- [ ] Print format generates PDF-ready HTML
- [ ] Stencil generates React/Vue/Angular wrappers
- [ ] All components pass unit tests
- [ ] Existing Go tests still pass (93 tests, 0 failures)
- [ ] Build pipeline: Stencil build -> copy to engine -> serve works
- [ ] Sample ERP (CRM + HRM) uses new components end-to-end
