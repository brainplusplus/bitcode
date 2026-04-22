# i18n Support for All Stencil Components — Design Document

**Date**: 2026-04-22
**Status**: Approved
**Scope**: `packages/components` — all ~94 Stencil web components

---

## 1. Problem Statement

All Stencil components currently render hardcoded English strings (~60+ unique strings across ~20 components). There is no RTL support, and date/number formatting is hardcoded to specific locales (`en-GB`, `id-ID`). An `i18n.ts` core exists (32 lines) but zero components consume it.

## 2. Goals

- Every component with user-facing text supports 11 languages out of the box
- Full RTL support for Arabic (icon mirroring, layout mirroring, CSS logical properties)
- Locale-aware date, number, currency, and relative-time formatting via `Intl` API with custom format override support
- Reactive locale switching — changing locale re-renders all components automatically
- Consumer apps can override/extend translations via `registerTranslations()`
- Fallback chain: current locale -> `en` -> key itself (never errors)

## 3. Supported Languages

| Code | Language | Direction | Plural Forms |
|------|----------|-----------|--------------|
| `en` | English | LTR | 2 (one, other) |
| `id` | Indonesian | LTR | 1 (other) |
| `fr` | French | LTR | 2 (one, other) |
| `de` | German | LTR | 2 (one, other) |
| `es` | Spanish | LTR | 2 (one, other) |
| `pt-BR` | Portuguese (Brazil) | LTR | 2 (one, other) |
| `ja` | Japanese | LTR | 1 (other) |
| `zh-CN` | Chinese Simplified | LTR | 1 (other) |
| `ko` | Korean | LTR | 1 (other) |
| `ar` | Arabic | **RTL** | 6 (zero, one, two, few, many, other) |
| `ru` | Russian | LTR | 3 (one, few, many) |

## 4. Architecture

### 4.1 Enhanced i18n Core

Enhance `packages/components/src/core/i18n.ts` from a simple singleton to a reactive, feature-complete i18n service.

```
i18n.ts (enhanced)
├── Reactive locale state (via @stencil/store)
│   └── Changing locale auto-triggers re-renders in all consuming components
├── Translation registry (bundled, all 11 locales)
│   └── en (fallback) + id, fr, de, es, pt-BR, ja, zh-CN, ko, ar, ru
├── t(key, params?) -> string translation with interpolation
├── tf.date(value, options?) -> locale-aware date formatting (Intl.DateTimeFormat)
├── tf.number(value, options?) -> locale-aware number formatting (Intl.NumberFormat)
├── tf.currency(value, currency, options?) -> locale-aware currency
├── tf.relativeTime(value, unit) -> relative time (Intl.RelativeTimeFormat)
├── plural(key, count, params?) -> ICU-style pluralization via Intl.PluralRules
├── dir -> 'rtl' | 'ltr' (auto-detected from locale)
├── isRTL -> boolean
├── setLocale(locale) -> switches locale + triggers re-renders
├── registerTranslations(locale, translations) -> extend/override at runtime
└── Fallback chain: current locale -> en -> key itself
```

**Reactivity mechanism**: Use `@stencil/store` to create a reactive `locale` state. When `setLocale()` is called, the store notifies all components that read `i18n.state.locale`, triggering re-renders automatically.

```typescript
import { createStore } from '@stencil/store';

const { state, onChange } = createStore({
  locale: 'en',
});

onChange('locale', (newLocale) => {
  // All components reading state.locale will re-render
});
```

### 4.2 Translation File Structure

```
packages/components/src/i18n/
├── index.ts              # Barrel: imports all locale JSONs, registers them
├── en.json               # English (fallback, always 100% complete)
├── id.json               # Indonesian
├── fr.json               # French
├── de.json               # German
├── es.json               # Spanish
├── pt-BR.json            # Portuguese (Brazil)
├── ja.json               # Japanese
├── zh-CN.json            # Chinese Simplified
├── ko.json               # Korean
├── ar.json               # Arabic
└── ru.json               # Russian
```

All 11 locale files are bundled into the component library (no lazy loading).

### 4.3 Translation Key Convention

Flat, dot-separated keys organized by component domain:

```json
{
  "common.loading": "Loading...",
  "common.save": "Save",
  "common.create": "Create",
  "common.cancel": "Cancel",
  "common.search": "Search...",
  "common.noResults": "No results",
  "common.prev": "Prev",
  "common.next": "Next",
  "common.page": "Page",
  "common.total": "Total",
  "common.delete": "Delete",
  "common.confirm": "Confirm",
  "common.copy": "Copy",
  "common.copied": "Copied",
  "common.records_one": "{count} record",
  "common.records_other": "{count} records",

  "datatable.filterColumn": "Filter column",
  "datatable.noRecords": "No records found",
  "datatable.show": "Show",
  "datatable.exportXls": "Export XLS",
  "datatable.presetName": "Preset name:",
  "datatable.presets": "Presets",
  "datatable.saveFilter": "Save Filter",

  "filter.addCondition": "+ Condition",
  "filter.addGroup": "+ Group",
  "filter.removeGroup": "Remove group",
  "filter.valuePlaceholder": "Value...",
  "filter.remove": "Remove",
  "filter.visual": "Visual",
  "filter.json": "JSON",
  "filter.applyJson": "Apply JSON",
  "filter.empty": "No filters. Click \"+ Condition\" to add.",

  "wizard.back": "Back",
  "wizard.next": "Next",
  "wizard.finish": "Finish",

  "quickentry.title": "Quick Create",
  "quickentry.saving": "Saving...",

  "report.title": "Report",
  "report.rows_one": "{count} row",
  "report.rows_other": "{count} rows",
  "report.exportCsv": "Export CSV",
  "report.average": "Average",

  "activity.title": "Activity",
  "activity.noActivities": "No activities yet",

  "timeline.title": "Change History",
  "timeline.noChanges": "No changes recorded",

  "calendar.loadingEvents": "Loading events...",

  "map.title": "Map",
  "map.locations": "locations",
  "map.loadingLocations": "Loading locations...",

  "handle.dragToReorder": "Drag to reorder",

  "barcode.placeholder": "Enter barcode value...",
  "barcode.qrError": "QR Error",
  "barcode.barcodeError": "Barcode Error"
}
```

### 4.4 Pluralization

Uses `Intl.PluralRules` to select the correct plural form per locale:

```typescript
plural(key: string, count: number, params?: Record<string, string | number>): string {
  const rule = new Intl.PluralRules(this.locale).select(count);
  // Try: key_zero, key_one, key_two, key_few, key_many, key_other
  const pluralKey = `${key}_${rule}`;
  const fallbackKey = `${key}_other`;
  return this.t(pluralKey, { count, ...params }) 
    || this.t(fallbackKey, { count, ...params });
}
```

Plural form counts per language:
- **1 form** (other only): id, ja, zh-CN, ko
- **2 forms** (one, other): en, fr, de, es, pt-BR
- **3 forms** (one, few, many): ru
- **6 forms** (zero, one, two, few, many, other): ar

## 5. RTL Support Strategy

### 5.1 Auto-detection

```typescript
const RTL_LOCALES = ['ar', 'he', 'fa', 'ur'];

get isRTL(): boolean {
  return RTL_LOCALES.includes(this.locale);
}

get dir(): 'rtl' | 'ltr' {
  return this.isRTL ? 'rtl' : 'ltr';
}
```

### 5.2 Host-level dir binding

Every component sets `dir` on its host element:

```tsx
@Element() el: HTMLElement;

componentWillRender() {
  this.el.dir = i18n.dir;
}
```

### 5.3 CSS Logical Properties Migration

Replace all physical CSS properties with logical equivalents:

| Physical (current) | Logical (new) |
|---------------------|---------------|
| `margin-left` | `margin-inline-start` |
| `margin-right` | `margin-inline-end` |
| `padding-left` | `padding-inline-start` |
| `padding-right` | `padding-inline-end` |
| `text-align: left` | `text-align: start` |
| `text-align: right` | `text-align: end` |
| `float: left` | `float: inline-start` |
| `left: 0` | `inset-inline-start: 0` |
| `right: 0` | `inset-inline-end: 0` |
| `border-left` | `border-inline-start` |
| `border-right` | `border-inline-end` |
| `border-radius: 4px 0 0 4px` | `border-start-start-radius: 4px; border-end-start-radius: 4px` |

### 5.4 Icon & Arrow Mirroring

```css
:host([dir="rtl"]) .icon-arrow-right,
:host([dir="rtl"]) .icon-chevron-right,
:host([dir="rtl"]) .icon-arrow-left,
:host([dir="rtl"]) .icon-chevron-left {
  transform: scaleX(-1);
}
```

### 5.5 Layout Mirroring

```css
:host([dir="rtl"]) .toolbar {
  flex-direction: row-reverse;
}

:host([dir="rtl"]) .pagination {
  flex-direction: row-reverse;
}

:host([dir="rtl"]) .sidebar-left {
  order: 2; /* moves to right side */
}
```

### 5.6 RTL Exceptions (do NOT flip)

- Phone numbers
- Code blocks / code editors
- Timestamps
- Progress bars (always left-to-right)
- LTR-embedded content (English text within Arabic UI)
- Mathematical expressions

## 6. Date/Number/Currency Formatting

### 6.1 Formatting API

```typescript
tf = {
  date(value: Date | string | number, options?: Intl.DateTimeFormatOptions): string {
    return new Intl.DateTimeFormat(this.locale, options).format(new Date(value));
  },

  number(value: number, options?: Intl.NumberFormatOptions): string {
    return new Intl.NumberFormat(this.locale, options).format(value);
  },

  currency(value: number, currency: string, options?: Intl.NumberFormatOptions): string {
    return new Intl.NumberFormat(this.locale, {
      style: 'currency',
      currency,
      ...options,
    }).format(value);
  },

  relativeTime(value: number, unit: Intl.RelativeTimeFormatUnit): string {
    return new Intl.RelativeTimeFormat(this.locale, { numeric: 'auto' }).format(value, unit);
  },
};
```

### 6.2 Migration from Hardcoded Formats

```typescript
// BEFORE (hardcoded):
new Date(v).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })

// AFTER (locale-aware with custom format support):
i18n.tf.date(v, { day: 'numeric', month: 'short', year: 'numeric' })
// Automatically uses current locale instead of hardcoded 'en-GB'

// BEFORE (hardcoded):
Intl.NumberFormat('id-ID', { style: 'currency', currency: fmt, maximumFractionDigits: 0 }).format(num)

// AFTER:
i18n.tf.currency(num, fmt, { maximumFractionDigits: 0 })
// Uses current locale instead of hardcoded 'id-ID'
```

## 7. Component Integration Pattern

Every component follows this pattern:

```tsx
import { Component, h, Element } from '@stencil/core';
import { i18n } from '../../core/i18n';

@Component({ tag: 'lc-datatable', styleUrl: 'lc-datatable.css', shadow: false })
export class LcDatatable {
  @Element() el: HTMLElement;

  componentWillRender() {
    this.el.dir = i18n.dir;
  }

  render() {
    return (
      <div>
        <span>{i18n.t('common.loading')}</span>
        <button>{i18n.t('datatable.exportXls')}</button>
        <span>{i18n.tf.date(someDate, { day: 'numeric', month: 'short' })}</span>
        <span>{i18n.tf.currency(amount, 'USD')}</span>
        <span>{i18n.plural('common.records', this.total)}</span>
      </div>
    );
  }
}
```

## 8. Consumer API

```typescript
import { i18n } from '@aspect/lc-components';

// Set locale at app startup
i18n.setLocale('ar');  // All components switch to Arabic + RTL

// Override/extend translations at runtime
i18n.registerTranslations('ar', {
  'common.save': 'حفظ',
  'custom.myAppKey': 'مفتاح مخصص'
});

// Read current state
console.log(i18n.locale);  // 'ar'
console.log(i18n.dir);     // 'rtl'
console.log(i18n.isRTL);   // true
```

## 9. Migration Priority

| Priority | Components | String Count | Effort |
|----------|-----------|-------------|--------|
| P0 (Critical) | lc-datatable, lc-filter-builder, lc-view-list | 15-20 each | High |
| P1 (High) | lc-view-form, lc-view-report, lc-dialog-quickentry, lc-dialog-wizard | 5-10 each | Medium |
| P2 (Medium) | lc-lookup-modal, lc-timeline, lc-view-activity, lc-search, lc-view-map | 3-5 each | Medium |
| P3 (Low) | lc-view-calendar, lc-view-kanban, lc-view-tree, lc-toast, lc-widget-handle, lc-field-barcode, lc-placeholder, lc-tabs | 1-3 each | Low |
| P4 (CSS only) | All ~94 components: physical CSS -> logical properties | CSS changes only | High (volume) |

## 10. Dependencies

- `@stencil/store` — for reactive locale state (likely already available or trivial to add)
- No external i18n library needed — everything built in-house using `Intl` API

## 11. Non-Goals

- Server-side rendering (SSR) i18n — out of scope for this design
- Automatic translation (machine translation) — translations are human-provided
- Module-level translations (CRM, HRM, Sales) — those remain separate, managed by the engine
- Lazy-loading of locale files — all bundled for simplicity
