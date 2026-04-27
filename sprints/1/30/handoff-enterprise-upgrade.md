# HANDOFF: Enterprise Component Upgrade

**Date:** 2026-07-28 (updated)
**Status:** Phase 1 ✅ + Phase 2 ✅ COMPLETE — all 34 field components upgraded
**Next session:** Phase 3-6 (select data features, datatable, charts, reactivity)

---

## CONTEXT — READ THESE FIRST

| Doc | Path | What |
|-----|------|------|
| Master design | `docs/plans/2026-07-27-enterprise-upgrade-master.md` | All architecture decisions, phase breakdown, doc structure |
| Phase 1 design | `docs/plans/2026-07-27-phase-1-core-infrastructure.md` | Core infra design (BcSetup, data-fetcher, validation, theming) |
| Phase 2 impl plan | `docs/plans/2026-07-27-phase-2-field-components.md` | Field component upgrade plan, groups, gaps |
| Matrix reference | `sprints/1/30/fix-1-22-matrix.md` | All 103 components current props/events/methods |
| Enterprise reference | `sprints/1/30/fix-1-22-enterprise.md` | Enterprise features wishlist |
| Component docs | `packages/components/docs/` | Core guides (theming, data-fetching, validation, reactivity, bc-setup) |

---

## WHAT'S DONE

### Phase 1: Core Infrastructure ✅ COMPLETE
All files created, built, documented, committed:
- `packages/components/src/core/bc-setup.ts` — Global config (auth, headers, theme, validators, reactivity API)
- `packages/components/src/core/data-fetcher.ts` — 4-level data fetching (local, URL, event intercept, custom fetcher)
- `packages/components/src/core/validation-engine.ts` — 3-level validation (built-in, custom JS, server-side)
- `packages/components/src/core/field-utils.ts` — Shared utilities (dirty/touched, ARIA, CSS classes, FormProxy, debounce)
- `packages/components/src/core/types.ts` — All event interfaces, FetchParams/FetchResult, ValidationResult, BcConfig
- `packages/components/src/global/global.css` — Size tokens, enterprise CSS vars, auto dark media query
- `packages/components/src/global/themes/dark.css` — Dark theme overrides
- `packages/components/docs/` — 7 core guide docs

### Phase 2: Field Components — 34/34 DONE ✅

| Group | Components | Status |
|-------|-----------|--------|
| A — Text Input (10) | string, password, integer, float, decimal, currency, percent, date, datetime, time | ✅ ALL DONE |
| B — Textarea (2) | text, smalltext | ✅ ALL DONE |
| C — Boolean/Choice (10) | checkbox, toggle, rating, color, radio, duration, multicheck, barcode, geo, signature | ✅ ALL DONE |
| D — Rich Editor (5) | richtext, markdown, html, code, json | ✅ ALL DONE |
| E — Data-driven (5) | select, link, dynlink, tags, tableselect | ✅ ALL DONE |
| F — File Upload (2) | file, image | ✅ ALL DONE |
| Docs | 34 component doc files in packages/components/docs/fields/ | ✅ ALL DONE |

---

## WHAT'S REMAINING — EXACT STEPS

### Step 1: Finish Group C (2 components)

**bc-field-geo** — Leaflet map. Currently `shadow: false`. Add enterprise wrapper around existing map logic. Preserve `componentDidLoad` map init, `disconnectedCallback` cleanup, click handler.

**bc-field-signature** — SignaturePad canvas. Currently `shadow: false`. Add enterprise wrapper. Preserve `componentDidLoad` pad init, `disconnectedCallback` cleanup, `endStroke` listener.

### Step 2: Group D — 5 Rich Editors

These use 3rd party editors. All currently `shadow: false`. Add enterprise wrapper preserving editor logic.

| Component | Editor | Key Logic to Preserve |
|-----------|--------|----------------------|
| bc-field-richtext | Tiptap | Editor init, toolbar, `cmd()` method |
| bc-field-markdown | CodeMirror | Editor init, preview toggle |
| bc-field-html | CodeMirror | Editor init, language extension |
| bc-field-code | CodeMirror | Editor init, language prop, `getLangExtension()` |
| bc-field-json | CodeMirror | Editor init, JSON validation |

### Step 3: Group E — 5 Data-driven (universal props only in Phase 2)

These have existing API client logic. Phase 2 only adds universal enterprise wrapper. Phase 3 adds 4-level data features.

| Component | Key Logic to Preserve |
|-----------|----------------------|
| bc-field-select | `getOptions()`, `handleChange()` — simple select |
| bc-field-link | `getApiClient()`, `search()`, `select()`, `clear()` — many2one |
| bc-field-dynlink | `getApiClient()`, model switching, `search()` |
| bc-field-tags | `getApiClient()`, `addTag()`, `removeTag()`, `search()` |
| bc-field-tableselect | `getApiClient()`, `addItem()`, `removeItem()`, `search()` |

### Step 4: Group F — 2 File Upload (universal props only)

| Component | Key Logic to Preserve |
|-----------|----------------------|
| bc-field-file | Drag/drop, upload, preview, file type validation |
| bc-field-image | Drag/drop, upload, image preview, crop |

### Step 5: Build & verify all
```bash
cd packages/components && npm run build
```

### Step 6: Generate 34 component doc files
Each in `packages/components/docs/fields/bc-field-*.md` following template in master design doc.

### Step 7: Update project docs
- `AGENTS.md` — component count, shadow DOM note
- `docs/codebase.md` — no new files, but note shadow:false change
- `docs/features.md` — update enterprise fields status

### Step 8: Commit & push

---

## PATTERN TO FOLLOW (proven, build-verified)

Every component upgrade follows this exact pattern. Use bc-field-string as reference:

### 1. Imports
```typescript
import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getAriaAttrs, getFieldClasses, getInputClasses, validateFieldValue, debounce } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';
```

### 2. Component decorator
```typescript
@Component({ tag: 'bc-field-xxx', styleUrl: 'bc-field-xxx.css', shadow: false })
```

### 3. Enterprise props to add (ALL components)
```typescript
@Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
@Prop({ mutable: true }) validationMessage: string = '';
@Prop() hint: string = '';
@Prop() size: 'sm' | 'md' | 'lg' = 'md';
@Prop() clearable: boolean = false;
@Prop() tooltip: string = '';
@Prop() loading: boolean = false;
@Prop() autofocus: boolean = false;
@Prop() defaultValue: <type> = <default>;
@Prop() validateOn: ValidateOn | '' = '';
```
Plus for text-based: `prefixText`, `suffixText`, `minLength`, `maxLength`, `showCount`, `pattern`, `patternMessage`
Plus for data-driven: `dependOn`, `dataSource`

### 4. Missing standard props to add where absent
- `required: boolean = false`
- `readonly: boolean = false`
- `placeholder: string = ''`

### 5. Internal state
```typescript
@State() private _fieldState: FieldState = createFieldState(<default>);
```

### 6. JS properties (not @Prop — set via JavaScript)
```typescript
customValidator?: (value: unknown) => string | null | Promise<string | null>;
validators?: Array<{ rule: string | ((value: unknown) => boolean | Promise<boolean>); message: string }>;
serverValidator?: string | ((value: unknown) => Promise<string | null>);
```

### 7. Events to add
```typescript
@Event() lcFieldFocus!: EventEmitter<FieldFocusEvent>;
@Event() lcFieldBlur!: EventEmitter<FieldBlurEvent>;
@Event() lcFieldClear!: EventEmitter<FieldClearEvent>;
@Event() lcFieldInvalid!: EventEmitter<FieldValidationEvent>;
@Event() lcFieldValid!: EventEmitter<FieldValidEvent>;
```

### 8. Methods to add
```typescript
@Method() async validate(): Promise<ValidationResult> { ... }
@Method() async reset(): Promise<void> { ... }
@Method() async clear(): Promise<void> { ... }
@Method() async setValue(value, emit = true): Promise<void> { ... }
@Method() async getValue(): Promise<type> { ... }
@Method() async focusField(): Promise<void> { ... }  // NOT focus() — reserved
@Method() async blurField(): Promise<void> { ... }   // NOT blur() — reserved
@Method() async isDirty(): Promise<boolean> { ... }
@Method() async isTouched(): Promise<boolean> { ... }
@Method() async setError(message): Promise<void> { ... }
@Method() async clearError(): Promise<void> { ... }
```

### 9. Lifecycle
```typescript
componentWillLoad() { this._fieldState = createFieldState(this.value || this.defaultValue); }
componentDidLoad() { if (this.autofocus) ... }
disconnectedCallback() { /* cleanup listeners */ }
```

### 10. Render wrapper
```typescript
const fieldClasses = getFieldClasses({ size, validationStatus, disabled, readonly, loading, dirty, touched });
const showError = this.validationStatus === 'invalid' && this.validationMessage;
const showHint = this.hint && !showError;

<div class={fieldClasses}>
  {label + required + tooltip}
  {/* existing render content */}
  <div class="bc-field-footer">
    {showError && <div class="bc-field-error" role="alert">{this.validationMessage}</div>}
    {showHint && <div class="bc-field-hint">{this.hint}</div>}
  </div>
</div>
```

---

## CRITICAL GOTCHAS (learned from implementation)

1. **`prefix` is reserved** in HTMLElement — use `prefixText` instead
2. **`suffix`** — use `suffixText` for consistency
3. **`focus()` is reserved** — use `focusField()`
4. **`blur()` is reserved** — use `blurField()`
5. **Unused imports cause build failure** — Stencil treats unused imports as errors
6. **`@State() private _errors`** — NOT needed. Use `validationMessage` prop instead
7. **`:host` works with `shadow: false`** — Stencil compiles to `[tag-name]` selector
8. **`getFieldClasses()` returns `Record<string, boolean>`** — use directly in JSX `class={}`
9. **Numeric fields clear to `0`**, string fields clear to `''`, boolean fields clear to `false`
10. **`componentWillLoad`** — capture initial value here, NOT in constructor

---

## AFTER PHASE 2 — REMAINING PHASES

| Phase | Scope | Design Doc |
|-------|-------|-----------|
| 3 | Select-family 4-level data, cascading, dependent | `phase-3-select-data-fields.md` (to create) |
| 4 | DataTable enterprise features | `phase-4-datatable.md` (to create) |
| 5 | Charts enterprise features | `phase-5-charts.md` (to create) |
| 6 | BcSetup.reactivity() impl, cross-field, server-side validation | `phase-6-reactivity-advanced.md` (to create) |

Phase 3-6 design docs need to be created. Master design doc has the specs.

---

## KEY DECISIONS (do not re-discuss)

1. **Standalone-first** — no BitCode dependency
2. **4-level data** — local, URL, event intercept, custom fetcher
3. **3-level validation** — built-in, custom JS, server-side
4. **Hybrid reactivity** — component props + BcSetup.reactivity() + FormEngine optional
5. **Theming** — light/dark/system/custom via CSS custom properties, no Tailwind
6. **Shadow DOM** — ALL `shadow: false`
7. **No engine Go changes** — ever
8. **Docs in `packages/components/docs/`** — self-contained for future repo split
9. **Motto** — Simple (usage), Flexible, Powerful, Complete (capability)
10. **Always** — think critically, detail, matang, lengkap, jujur

---

**To resume:** Read this file first, then continue from "Step 1: Finish Group C" above.
