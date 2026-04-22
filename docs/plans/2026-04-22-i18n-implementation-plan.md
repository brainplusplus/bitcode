# i18n Stencil Components — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add full i18n support (11 languages, RTL, Intl formatting, pluralization) to all ~94 Stencil web components in `packages/components`.

**Architecture:** Enhance the existing `packages/components/src/core/i18n.ts` singleton with `@stencil/store` reactivity. Bundle all 11 locale JSON files. Migrate hardcoded strings to `i18n.t()` calls. Add CSS logical properties and RTL layout mirroring for Arabic.

**Tech Stack:** Stencil.js, @stencil/store, Intl API (DateTimeFormat, NumberFormat, PluralRules, RelativeTimeFormat)

**Design Doc:** `docs/plans/2026-04-22-i18n-stencil-components-design.md`

---

## Task 1: Install @stencil/store dependency

**Files:**
- Modify: `packages/components/package.json`

**Step 1: Install the package**

Run:
```bash
cd packages/components && npm install @stencil/store
```

Expected: `@stencil/store` added to `dependencies` in `package.json`.

**Step 2: Verify installation**

Run:
```bash
cd packages/components && node -e "require('@stencil/store')"
```

Expected: No error.

**Step 3: Commit**

```bash
git add packages/components/package.json packages/components/package-lock.json
git commit -m "chore: add @stencil/store for reactive i18n state"
```

---

## Task 2: Create the enhanced i18n core

**Files:**
- Modify: `packages/components/src/core/i18n.ts` (rewrite entirely)

**Step 1: Write the enhanced i18n.ts**

Replace the entire contents of `packages/components/src/core/i18n.ts` with:

```typescript
import { createStore } from '@stencil/store';

// --- Types ---

export type Locale = 'en' | 'id' | 'fr' | 'de' | 'es' | 'pt-BR' | 'ja' | 'zh-CN' | 'ko' | 'ar' | 'ru';
export type Translations = Record<string, string>;
export type Direction = 'ltr' | 'rtl';

// --- Constants ---

const RTL_LOCALES: ReadonlySet<string> = new Set(['ar', 'he', 'fa', 'ur']);

const SUPPORTED_LOCALES: readonly Locale[] = [
  'en', 'id', 'fr', 'de', 'es', 'pt-BR', 'ja', 'zh-CN', 'ko', 'ar', 'ru',
] as const;

// --- Reactive Store ---

const { state, onChange } = createStore<{ locale: Locale }>({
  locale: 'en',
});

// --- Translation Registry ---

const registry = new Map<string, Translations>();

// --- Core Class ---

class I18n {

  /** All supported locale codes */
  readonly supportedLocales: readonly Locale[] = SUPPORTED_LOCALES;

  // --- Locale ---

  get locale(): Locale {
    return state.locale;
  }

  setLocale(locale: Locale): void {
    if (!SUPPORTED_LOCALES.includes(locale)) {
      console.warn(`[i18n] Unsupported locale "${locale}". Falling back to "en".`);
      state.locale = 'en';
      return;
    }
    state.locale = locale;
  }

  /** Subscribe to locale changes. Returns unsubscribe function. */
  onLocaleChange(callback: (locale: Locale) => void): () => void {
    return onChange('locale', callback);
  }

  // --- Direction ---

  get dir(): Direction {
    return RTL_LOCALES.has(state.locale) ? 'rtl' : 'ltr';
  }

  get isRTL(): boolean {
    return RTL_LOCALES.has(state.locale);
  }

  // --- Translation Registration ---

  /**
   * Register translations for a locale. Merges with existing.
   * Can be called multiple times (e.g., app-level overrides).
   */
  registerTranslations(locale: string, translations: Translations): void {
    const existing = registry.get(locale) || {};
    registry.set(locale, { ...existing, ...translations });
  }

  // --- Translation Lookup ---

  /**
   * Translate a key with optional interpolation params.
   * Fallback chain: current locale -> 'en' -> key itself.
   *
   * Interpolation: {paramName} in translation string replaced by params.paramName
   */
  t(key: string, params?: Record<string, string | number>): string {
    const currentDict = registry.get(state.locale);
    const fallbackDict = registry.get('en');

    let text = currentDict?.[key] ?? fallbackDict?.[key] ?? key;

    if (params) {
      for (const [k, v] of Object.entries(params)) {
        text = text.replace(new RegExp(`\\{${k}\\}`, 'g'), String(v));
      }
    }

    return text;
  }

  // --- Pluralization ---

  /**
   * Pluralize a key using Intl.PluralRules.
   *
   * Looks up: key_zero, key_one, key_two, key_few, key_many, key_other
   * Falls back to key_other, then key itself.
   *
   * The {count} param is automatically injected.
   */
  plural(key: string, count: number, params?: Record<string, string | number>): string {
    const rule = new Intl.PluralRules(state.locale).select(count);
    const mergedParams = { count, ...params };

    // Try exact plural form first
    const pluralKey = `${key}_${rule}`;
    const currentDict = registry.get(state.locale);
    const fallbackDict = registry.get('en');

    if (currentDict?.[pluralKey] || fallbackDict?.[pluralKey]) {
      return this.t(pluralKey, mergedParams);
    }

    // Fallback to _other
    const otherKey = `${key}_other`;
    if (currentDict?.[otherKey] || fallbackDict?.[otherKey]) {
      return this.t(otherKey, mergedParams);
    }

    // Final fallback: key itself
    return this.t(key, mergedParams);
  }

  // --- Formatting (Intl API) ---

  readonly tf = {
    /**
     * Format a date value using Intl.DateTimeFormat.
     * Uses current locale. Custom options override defaults.
     */
    date: (value: Date | string | number, options?: Intl.DateTimeFormatOptions): string => {
      try {
        const date = value instanceof Date ? value : new Date(value);
        return new Intl.DateTimeFormat(state.locale, options).format(date);
      } catch {
        return String(value);
      }
    },

    /**
     * Format a number using Intl.NumberFormat.
     * Uses current locale. Custom options override defaults.
     */
    number: (value: number, options?: Intl.NumberFormatOptions): string => {
      try {
        return new Intl.NumberFormat(state.locale, options).format(value);
      } catch {
        return String(value);
      }
    },

    /**
     * Format a currency value using Intl.NumberFormat.
     * Uses current locale. Currency code required (e.g., 'USD', 'IDR').
     */
    currency: (value: number, currency: string, options?: Intl.NumberFormatOptions): string => {
      try {
        return new Intl.NumberFormat(state.locale, {
          style: 'currency',
          currency,
          ...options,
        }).format(value);
      } catch {
        return String(value);
      }
    },

    /**
     * Format a relative time using Intl.RelativeTimeFormat.
     * e.g., tf.relativeTime(-1, 'day') -> "yesterday" (en) / "kemarin" (id)
     */
    relativeTime: (value: number, unit: Intl.RelativeTimeFormatUnit): string => {
      try {
        return new Intl.RelativeTimeFormat(state.locale, { numeric: 'auto' }).format(value, unit);
      } catch {
        return String(value);
      }
    },
  };
}

export const i18n = new I18n();
```

**Step 2: Verify TypeScript compiles**

Run:
```bash
cd packages/components && npx stencil build --no-open 2>&1 | head -20
```

Expected: No TypeScript errors related to `i18n.ts`.

**Step 3: Commit**

```bash
git add packages/components/src/core/i18n.ts
git commit -m "feat(i18n): rewrite i18n core with reactive store, Intl formatting, and pluralization"
```

---

## Task 3: Create English (en) translation file — the master key set

**Files:**
- Create: `packages/components/src/i18n/en.json`

**Step 1: Create the i18n directory and en.json**

Create `packages/components/src/i18n/en.json` with the complete English translation keys. This is the master set — all other locale files must have the same keys.

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
  "common.of": "of",
  "common.records_one": "{count} record",
  "common.records_other": "{count} records",
  "common.back": "Back",
  "common.finish": "Finish",
  "common.close": "Close",
  "common.ok": "OK",
  "common.yes": "Yes",
  "common.no": "No",
  "common.edit": "Edit",
  "common.actions": "Actions",

  "datatable.filterColumn": "Filter column",
  "datatable.noRecords": "No records found",
  "datatable.show": "Show",
  "datatable.exportXls": "Export XLS",
  "datatable.presetName": "Preset name:",
  "datatable.presets": "Presets",
  "datatable.saveFilter": "Save Filter",
  "datatable.deleteSelected": "Delete Selected",
  "datatable.columns": "Columns",

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
  "activity.activities_one": "{count} activity",
  "activity.activities_other": "{count} activities",
  "activity.noActivities": "No activities yet",

  "timeline.title": "Change History",
  "timeline.noChanges": "No changes recorded",

  "calendar.loadingEvents": "Loading events...",

  "map.title": "Map",
  "map.locations_one": "{count} location",
  "map.locations_other": "{count} locations",
  "map.loadingLocations": "Loading locations...",

  "handle.dragToReorder": "Drag to reorder",

  "barcode.placeholder": "Enter barcode value...",
  "barcode.qrError": "QR Error",
  "barcode.barcodeError": "Barcode Error",

  "placeholder.default": "BitCode Component",

  "tab.default": "Tab",

  "form.saving": "Saving...",

  "export.title": "Export",

  "print.title": "Print",

  "toast.dismiss": "Dismiss",

  "confirm.title": "Confirm",
  "confirm.message": "Are you sure?"
}
```

**Step 2: Commit**

```bash
git add packages/components/src/i18n/en.json
git commit -m "feat(i18n): add English master translation file"
```

---

## Task 4: Create all 10 remaining locale translation files

**Files:**
- Create: `packages/components/src/i18n/id.json`
- Create: `packages/components/src/i18n/fr.json`
- Create: `packages/components/src/i18n/de.json`
- Create: `packages/components/src/i18n/es.json`
- Create: `packages/components/src/i18n/pt-BR.json`
- Create: `packages/components/src/i18n/ja.json`
- Create: `packages/components/src/i18n/zh-CN.json`
- Create: `packages/components/src/i18n/ko.json`
- Create: `packages/components/src/i18n/ar.json`
- Create: `packages/components/src/i18n/ru.json`

**Step 1: Create each locale file**

Each file must have the EXACT same keys as `en.json` but with translated values. Use professional-quality translations (not machine-translated placeholders).

Key notes per locale:
- **id.json** (Indonesian): 1 plural form only (`_other`). No `_one` needed.
- **ja.json** (Japanese): 1 plural form only (`_other`). No `_one` needed.
- **zh-CN.json** (Chinese Simplified): 1 plural form only (`_other`). No `_one` needed.
- **ko.json** (Korean): 1 plural form only (`_other`). No `_one` needed.
- **ru.json** (Russian): 3 plural forms needed (`_one`, `_few`, `_many`). Example:
  - `"common.records_one": "{count} запись"`
  - `"common.records_few": "{count} записи"`
  - `"common.records_many": "{count} записей"`
- **ar.json** (Arabic): 6 plural forms needed (`_zero`, `_one`, `_two`, `_few`, `_many`, `_other`). Example:
  - `"common.records_zero": "لا سجلات"`
  - `"common.records_one": "سجل واحد"`
  - `"common.records_two": "سجلان"`
  - `"common.records_few": "{count} سجلات"`
  - `"common.records_many": "{count} سجلاً"`
  - `"common.records_other": "{count} سجل"`
- **fr.json, de.json, es.json, pt-BR.json**: 2 plural forms (`_one`, `_other`).

**Step 2: Commit**

```bash
git add packages/components/src/i18n/*.json
git commit -m "feat(i18n): add translation files for all 11 locales"
```

---

## Task 5: Create the i18n barrel (index.ts) that registers all locales

**Files:**
- Create: `packages/components/src/i18n/index.ts`

**Step 1: Write the barrel file**

Create `packages/components/src/i18n/index.ts`:

```typescript
import { i18n } from '../core/i18n';

import en from './en.json';
import id from './id.json';
import fr from './fr.json';
import de from './de.json';
import es from './es.json';
import ptBR from './pt-BR.json';
import ja from './ja.json';
import zhCN from './zh-CN.json';
import ko from './ko.json';
import ar from './ar.json';
import ru from './ru.json';

// Register all bundled translations
i18n.registerTranslations('en', en);
i18n.registerTranslations('id', id);
i18n.registerTranslations('fr', fr);
i18n.registerTranslations('de', de);
i18n.registerTranslations('es', es);
i18n.registerTranslations('pt-BR', ptBR);
i18n.registerTranslations('ja', ja);
i18n.registerTranslations('zh-CN', zhCN);
i18n.registerTranslations('ko', ko);
i18n.registerTranslations('ar', ar);
i18n.registerTranslations('ru', ru);

export { i18n };
```

**Step 2: Add JSON module resolution to tsconfig**

Modify `packages/components/tsconfig.json` — ensure `resolveJsonModule` and `esModuleInterop` are enabled:

```json
{
  "compilerOptions": {
    "resolveJsonModule": true,
    "esModuleInterop": true
  }
}
```

Only add these keys if they are not already present. Do not remove existing keys.

**Step 3: Wire the i18n barrel as a global script in stencil.config.ts**

Modify `packages/components/stencil.config.ts` to add the `globalScript` property so translations are registered before any component renders:

```typescript
import { Config } from '@stencil/core';

export const config: Config = {
  namespace: 'lc-components',
  globalScript: 'src/i18n/index.ts',   // <-- ADD THIS LINE
  outputTargets: [
    // ... existing targets unchanged
  ],
  globalStyle: 'src/global/global.css',
  testing: {
    browserHeadless: 'new',
  },
};
```

**Step 4: Verify build**

Run:
```bash
cd packages/components && npx stencil build
```

Expected: Build succeeds. All locale JSONs are bundled.

**Step 5: Commit**

```bash
git add packages/components/src/i18n/index.ts packages/components/tsconfig.json packages/components/stencil.config.ts
git commit -m "feat(i18n): wire i18n barrel as global script, register all 11 locales at startup"
```

---

## Task 6: Migrate P0 components — lc-datatable

**Files:**
- Modify: `packages/components/src/components/datatable/lc-datatable/lc-datatable.tsx`
- Modify: `packages/components/src/components/datatable/lc-datatable/lc-datatable.css`

**Step 1: Replace hardcoded strings with i18n.t() calls**

In `lc-datatable.tsx`:

1. Add import at top: `import { i18n } from '../../../core/i18n';`
2. Add RTL binding in `componentWillRender()`:
   ```typescript
   componentWillRender() {
     this.el.dir = i18n.dir;
   }
   ```
   If `componentWillRender` doesn't exist, add it.
3. Replace every hardcoded string:
   - `"Loading..."` -> `{i18n.t('common.loading')}`
   - `"Export XLS"` -> `{i18n.t('datatable.exportXls')}`
   - `"Filter column"` (title attr) -> `title={i18n.t('datatable.filterColumn')}`
   - `"No records found"` -> `{i18n.t('datatable.noRecords')}`
   - `"Show"` -> `{i18n.t('datatable.show')}`
   - `"Page"` -> `{i18n.t('common.page')}`
   - `"Total"` -> `{i18n.t('common.total')}`
   - `"records"` / `${this.total} records` -> `{i18n.plural('common.records', this.total)}`
   - `"Preset name:"` -> `{i18n.t('datatable.presetName')}`
   - `"Presets"` -> `{i18n.t('datatable.presets')}`
   - `"Save Filter"` -> `{i18n.t('datatable.saveFilter')}`
   - `"Delete Selected"` -> `{i18n.t('datatable.deleteSelected')}`
   - `"Columns"` -> `{i18n.t('datatable.columns')}`
   - `"Filter"` -> `{i18n.t('filter.addCondition')}` or appropriate key
4. Replace hardcoded date formatting:
   - `new Date(v).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })` -> `i18n.tf.date(v, { day: 'numeric', month: 'short', year: 'numeric' })`
5. Replace hardcoded currency formatting:
   - `Intl.NumberFormat('id-ID', { style: 'currency', currency: fmt, maximumFractionDigits: 0 }).format(num)` -> `i18n.tf.currency(num, fmt, { maximumFractionDigits: 0 })`
6. Replace hardcoded number formatting:
   - `.toLocaleString()` -> `i18n.tf.number(value)`

**Step 2: Migrate CSS to logical properties**

In `lc-datatable.css`:
- Replace `margin-left` -> `margin-inline-start`
- Replace `margin-right` -> `margin-inline-end`
- Replace `padding-left` -> `padding-inline-start`
- Replace `padding-right` -> `padding-inline-end`
- Replace `text-align: left` -> `text-align: start`
- Replace `text-align: right` -> `text-align: end`
- Replace `left:` -> `inset-inline-start:` (for positioned elements)
- Replace `right:` -> `inset-inline-end:` (for positioned elements)
- Replace `border-left` -> `border-inline-start`
- Replace `border-right` -> `border-inline-end`
- Add RTL-specific overrides for flex layouts:
  ```css
  [dir="rtl"] .lc-datatable .pagination {
    flex-direction: row-reverse;
  }
  ```

**Step 3: Verify build**

Run:
```bash
cd packages/components && npx stencil build
```

Expected: No errors.

**Step 4: Commit**

```bash
git add packages/components/src/components/datatable/lc-datatable/
git commit -m "feat(i18n): migrate lc-datatable to i18n.t() with RTL support"
```

---

## Task 7: Migrate P0 components — lc-filter-builder

**Files:**
- Modify: `packages/components/src/components/datatable/lc-filter-builder/lc-filter-builder.tsx`
- Modify: `packages/components/src/components/datatable/lc-filter-builder/lc-filter-builder.css` (if exists)

**Step 1: Replace hardcoded strings**

Same pattern as Task 6. Key replacements:
- `"+ Condition"` -> `{i18n.t('filter.addCondition')}`
- `"+ Group"` -> `{i18n.t('filter.addGroup')}`
- `"Remove group"` -> `{i18n.t('filter.removeGroup')}`
- `"Value..."` -> `placeholder={i18n.t('filter.valuePlaceholder')}`
- `"Remove"` -> `{i18n.t('filter.remove')}`
- `"Visual"` -> `{i18n.t('filter.visual')}`
- `"JSON"` -> `{i18n.t('filter.json')}`
- `"Apply JSON"` -> `{i18n.t('filter.applyJson')}`
- `"No filters..."` -> `{i18n.t('filter.empty')}`

Add `componentWillRender` with `this.el.dir = i18n.dir;`

**Step 2: CSS logical properties migration (same pattern as Task 6)**

**Step 3: Verify build, commit**

```bash
git add packages/components/src/components/datatable/lc-filter-builder/
git commit -m "feat(i18n): migrate lc-filter-builder to i18n.t() with RTL support"
```

---

## Task 8: Migrate P0 components — lc-view-list

**Files:**
- Modify: `packages/components/src/components/views/lc-view-list/lc-view-list.tsx`
- Modify: `packages/components/src/components/views/lc-view-list/lc-view-list.css` (if exists)

**Step 1: Replace hardcoded strings**

Key replacements:
- `"Search..."` -> `placeholder={i18n.t('common.search')}`
- `"Loading..."` -> `{i18n.t('common.loading')}`
- `"Prev"` -> `{i18n.t('common.prev')}`
- `"Next"` -> `{i18n.t('common.next')}`
- `"Page"` -> `{i18n.t('common.page')}`
- `${this.total} records` -> `{i18n.plural('common.records', this.total)}`
- `Page X of Y` -> `{i18n.t('common.page')} {this.page} {i18n.t('common.of')} {totalPages}`

Add `componentWillRender` with RTL binding. CSS logical properties migration.

**Step 2: Verify build, commit**

```bash
git add packages/components/src/components/views/lc-view-list/
git commit -m "feat(i18n): migrate lc-view-list to i18n.t() with RTL support"
```

---

## Task 9: Migrate P1 components (batch)

**Files:**
- Modify: `packages/components/src/components/views/lc-view-form/lc-view-form.tsx` + `.css`
- Modify: `packages/components/src/components/views/lc-view-report/lc-view-report.tsx` + `.css`
- Modify: `packages/components/src/components/dialogs/lc-dialog-quickentry/lc-dialog-quickentry.tsx` + `.css`
- Modify: `packages/components/src/components/dialogs/lc-dialog-wizard/lc-dialog-wizard.tsx` + `.css`

**Step 1: For each component, apply the same pattern:**

1. Add `import { i18n } from '../../../core/i18n';` (adjust relative path)
2. Add `componentWillRender() { this.el.dir = i18n.dir; }`
3. Replace all hardcoded strings with `i18n.t()` calls using the keys from `en.json`
4. Replace date/number formatting with `i18n.tf.*` calls
5. Migrate CSS to logical properties

Key replacements per component:

**lc-view-form:**
- `"Loading..."` -> `i18n.t('common.loading')`
- `"Save"` -> `i18n.t('common.save')`
- `"Create"` -> `i18n.t('common.create')`

**lc-view-report:**
- `"Report"` -> `i18n.t('report.title')`
- `"rows"` -> `i18n.plural('report.rows', count)`
- `"Export CSV"` -> `i18n.t('report.exportCsv')`
- `"Loading..."` -> `i18n.t('common.loading')`
- `"Total"` -> `i18n.t('common.total')`
- `"Average"` -> `i18n.t('report.average')`
- `.toLocaleString()` -> `i18n.tf.number(value)`

**lc-dialog-quickentry:**
- `"Quick Create"` -> `i18n.t('quickentry.title')`
- `"Cancel"` -> `i18n.t('common.cancel')`
- `"Create"` -> `i18n.t('common.create')`
- `"Saving..."` -> `i18n.t('quickentry.saving')`

**lc-dialog-wizard:**
- `"Back"` -> `i18n.t('wizard.back')`
- `"Next"` -> `i18n.t('wizard.next')`
- `"Finish"` -> `i18n.t('wizard.finish')`

**Step 2: Verify build**

```bash
cd packages/components && npx stencil build
```

**Step 3: Commit**

```bash
git add packages/components/src/components/views/lc-view-form/ \
       packages/components/src/components/views/lc-view-report/ \
       packages/components/src/components/dialogs/lc-dialog-quickentry/ \
       packages/components/src/components/dialogs/lc-dialog-wizard/
git commit -m "feat(i18n): migrate P1 components (form, report, quickentry, wizard)"
```

---

## Task 10: Migrate P2 components (batch)

**Files:**
- Modify: `packages/components/src/components/datatable/lc-lookup-modal/lc-lookup-modal.tsx` + `.css`
- Modify: `packages/components/src/components/social/lc-timeline/lc-timeline.tsx` + `.css`
- Modify: `packages/components/src/components/views/lc-view-activity/lc-view-activity.tsx` + `.css`
- Modify: `packages/components/src/components/search/lc-search/lc-search.tsx` + `.css`
- Modify: `packages/components/src/components/views/lc-view-map/lc-view-map.tsx` + `.css`

**Step 1: Same pattern as previous tasks**

Key replacements per component:

**lc-lookup-modal:** `"Search..."`, `"Loading..."`, `"No results"`, `"{total} results"`
**lc-timeline:** `"Change History"`, `"Loading..."`, `"No changes recorded"`, date formatting
**lc-view-activity:** `"Activity"`, `"activities"`, `"Loading..."`, `"No activities yet"`, date formatting
**lc-search:** `"Search..."` placeholder
**lc-view-map:** `"Map"`, `"locations"`, `"Loading locations..."`

Add RTL binding + CSS logical properties to each.

**Step 2: Verify build, commit**

```bash
git add packages/components/src/components/datatable/lc-lookup-modal/ \
       packages/components/src/components/social/lc-timeline/ \
       packages/components/src/components/views/lc-view-activity/ \
       packages/components/src/components/search/lc-search/ \
       packages/components/src/components/views/lc-view-map/
git commit -m "feat(i18n): migrate P2 components (lookup, timeline, activity, search, map)"
```

---

## Task 11: Migrate P3 components (batch)

**Files:**
- Modify: `packages/components/src/components/views/lc-view-calendar/lc-view-calendar.tsx` + `.css`
- Modify: `packages/components/src/components/views/lc-view-kanban/lc-view-kanban.tsx` + `.css`
- Modify: `packages/components/src/components/views/lc-view-tree/lc-view-tree.tsx` + `.css`
- Modify: `packages/components/src/components/dialogs/lc-toast/lc-toast.tsx` + `.css`
- Modify: `packages/components/src/components/widgets/lc-widget-handle/lc-widget-handle.tsx`
- Modify: `packages/components/src/components/fields/lc-field-barcode/lc-field-barcode.tsx`
- Modify: `packages/components/src/components/lc-placeholder/lc-placeholder.tsx`
- Modify: `packages/components/src/components/layout/lc-tabs/lc-tabs.tsx`
- Modify: `packages/components/src/components/layout/lc-tab/lc-tab.tsx`

**Step 1: Same pattern. These have 1-3 strings each — quick migrations.**

**Step 2: Verify build, commit**

```bash
git add packages/components/src/components/views/lc-view-calendar/ \
       packages/components/src/components/views/lc-view-kanban/ \
       packages/components/src/components/views/lc-view-tree/ \
       packages/components/src/components/dialogs/lc-toast/ \
       packages/components/src/components/widgets/lc-widget-handle/ \
       packages/components/src/components/fields/lc-field-barcode/ \
       packages/components/src/components/lc-placeholder/ \
       packages/components/src/components/layout/lc-tabs/ \
       packages/components/src/components/layout/lc-tab/
git commit -m "feat(i18n): migrate P3 components (calendar, kanban, tree, toast, handle, barcode, placeholder, tabs)"
```

---

## Task 12: RTL CSS migration for remaining components (P4)

**Files:**
- Modify: ALL `.css` files under `packages/components/src/components/`

**Step 1: Audit all CSS files for physical properties**

Search all `.css` files for:
- `margin-left`, `margin-right`
- `padding-left`, `padding-right`
- `text-align: left`, `text-align: right`
- `float: left`, `float: right`
- `left:`, `right:` (positioned elements)
- `border-left`, `border-right`
- `border-radius` with asymmetric values

**Step 2: Replace with logical equivalents**

Use the mapping from the design doc (Section 5.3). For each CSS file:
- `margin-left` -> `margin-inline-start`
- `margin-right` -> `margin-inline-end`
- `padding-left` -> `padding-inline-start`
- `padding-right` -> `padding-inline-end`
- `text-align: left` -> `text-align: start`
- `text-align: right` -> `text-align: end`
- `left:` -> `inset-inline-start:` (for positioned elements)
- `right:` -> `inset-inline-end:` (for positioned elements)
- `border-left` -> `border-inline-start`
- `border-right` -> `border-inline-end`

**Step 3: Add RTL-specific overrides where needed**

For components with flex layouts, add `[dir="rtl"]` overrides for `flex-direction: row-reverse` where the visual order must flip.

**Step 4: Add `componentWillRender` RTL binding to ALL remaining components**

Every component that doesn't already have it gets:
```typescript
import { i18n } from '../../core/i18n';  // adjust path

componentWillRender() {
  this.el.dir = i18n.dir;
}
```

**Step 5: Verify build**

```bash
cd packages/components && npx stencil build
```

**Step 6: Commit**

```bash
git add packages/components/src/components/
git commit -m "feat(i18n): migrate all component CSS to logical properties + RTL dir binding"
```

---

## Task 13: Export i18n from the package public API

**Files:**
- Modify: `packages/components/src/index.ts` (or wherever the package barrel export is)

**Step 1: Add i18n export**

Ensure the consuming app can import `i18n` from the package:

```typescript
export { i18n } from './core/i18n';
export type { Locale, Translations, Direction } from './core/i18n';
```

**Step 2: Verify build, commit**

```bash
cd packages/components && npx stencil build
git add packages/components/src/
git commit -m "feat(i18n): export i18n API from package public surface"
```

---

## Task 14: Final build verification and smoke test

**Step 1: Full clean build**

```bash
cd packages/components && rm -rf dist www .stencil && npx stencil build
```

Expected: Build succeeds with zero errors.

**Step 2: Verify bundle includes translations**

Check that the built output contains translation strings:
```bash
grep -r "common.loading" packages/components/dist/ | head -5
```

Expected: Translation keys appear in the bundle.

**Step 3: Verify no remaining hardcoded strings**

Search for known hardcoded strings that should have been migrated:
```bash
grep -rn '"Loading\.\.\."' packages/components/src/components/ --include="*.tsx"
grep -rn "'Loading\.\.\.'" packages/components/src/components/ --include="*.tsx"
grep -rn '"No records found"' packages/components/src/components/ --include="*.tsx"
grep -rn "'en-GB'" packages/components/src/components/ --include="*.tsx"
grep -rn "'id-ID'" packages/components/src/components/ --include="*.tsx"
```

Expected: Zero matches for all searches (all migrated to i18n.t() / i18n.tf.*).

**Step 4: Commit (if any fixes needed)**

```bash
git add -A && git commit -m "fix(i18n): address remaining hardcoded strings found in verification"
```

---

## Summary

| Task | Description | Effort |
|------|-------------|--------|
| 1 | Install @stencil/store | 5 min |
| 2 | Enhanced i18n core (reactive, Intl, plural) | 30 min |
| 3 | English master translation file | 15 min |
| 4 | 10 remaining locale files (id, fr, de, es, pt-BR, ja, zh-CN, ko, ar, ru) | 2-3 hrs |
| 5 | i18n barrel + stencil config wiring | 15 min |
| 6 | Migrate lc-datatable (P0) | 45 min |
| 7 | Migrate lc-filter-builder (P0) | 30 min |
| 8 | Migrate lc-view-list (P0) | 30 min |
| 9 | Migrate P1 components (4 components) | 1 hr |
| 10 | Migrate P2 components (5 components) | 45 min |
| 11 | Migrate P3 components (9 components) | 30 min |
| 12 | RTL CSS migration for all components | 2-3 hrs |
| 13 | Export i18n from public API | 10 min |
| 14 | Final verification | 15 min |
| **Total** | | **~9-11 hrs** |
