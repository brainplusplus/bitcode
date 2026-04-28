# Changelog

All notable changes to `@bitcode/components` will be documented in this file.

## [0.2.0] - 2026-07-28

### Enterprise Component Upgrade

Complete enterprise upgrade of all 103 components. Every component is now standalone-capable — works without BitCode framework.

### Added

#### Core Infrastructure
- **BcSetup** — Global configuration singleton (auth, headers, theme, validators, reactivity rules)
- **data-fetcher** — 4-level data fetching: local data, URL endpoint, event intercept, custom fetcher function
- **validation-engine** — 3-level validation pipeline: built-in rules, custom JS validators, server-side validation
- **field-utils** — Shared utilities: dirty/touched tracking, ARIA attributes, CSS classes, FormProxy, debounce
- **Barrel exports** — `import { BcSetup, validateAllFields } from '@bitcode/components/utils'`

#### Theming
- Light theme (default)
- Dark theme (`data-bc-theme="dark"` or `BcSetup.configure({ theme: 'dark' })`)
- System preference detection (`BcSetup.configure({ theme: 'system' })`)
- Custom themes via CSS custom properties
- Size tokens: `sm`, `md`, `lg`
- CSS variables for validation states, dropdowns, tables, charts, tooltips, tags

#### 34 Field Components
- Enterprise props: `validationStatus`, `validationMessage`, `hint`, `size`, `clearable`, `tooltip`, `loading`, `autofocus`, `defaultValue`, `validateOn`, `prefixText`, `suffixText`
- Enterprise events: `lcFieldFocus`, `lcFieldBlur`, `lcFieldClear`, `lcFieldInvalid`, `lcFieldValid`
- Enterprise methods: `validate()`, `reset()`, `clear()`, `setValue()`, `getValue()`, `focusField()`, `blurField()`, `isDirty()`, `isTouched()`, `setError()`, `clearError()`
- Dirty/touched/pristine state tracking
- ARIA attributes auto-generated
- 3-level validation (built-in props, custom JS, server-side)

#### bc-field-select
- Custom searchable dropdown (replaces native `<select>`)
- 4-level data fetching with `dataSource` prop
- Cascading via `dependOn` + `dataSource` with `{field}` placeholders
- Server-side search with debounce
- Multiple select with tag display and individual remove
- Creatable options
- Keyboard navigation (arrow keys, enter, escape)
- Virtual scroll pagination (100 at a time, "Show more" button)
- Events: `lcOptionsLoad`, `lcOptionsError`, `lcOptionCreate`, `lcDropdownOpen`, `lcDropdownClose`
- Methods: `loadOptions()`, `reloadOptions()`, `getOptions()`, `getSelectedOptions()`, `setOptions()`, `open()`, `close()`

#### bc-field-link, bc-field-dynlink, bc-field-tags, bc-field-tableselect
- Replaced `getApiClient()` with `fetchOptions()` — standalone-capable
- `dataSource` prop for custom API endpoints
- `fetchHeaders` prop for custom headers

#### bc-datatable
- 4-level data fetching: `localData`, `dataSource`, `lcBeforeFetch`/`lcAfterFetch`, `dataFetcher`
- Methods: `refresh()`, `getData()`, `setData()`, `getSelected()`, `clearSelection()`, `selectAll()`, `goToPage()`, `sortBy()`, `exportCSV()`, `scrollToRow()`
- Events: `lcPageChange`, `lcSortChange`, `lcFilterChange`
- Auth headers auto-included via BcSetup

#### 11 Chart Components
- Enterprise props: `colors`, `legend`, `tooltipEnabled`, `animate`, `height`, `loading`, `dataSource`, `fetchHeaders`, `refreshInterval`
- Event: `lcChartClick`
- Methods: `updateData()`, `setData()`, `refresh()`, `resize()`, `exportImage()`
- Auto-refresh interval with `dataFetcher` support

#### 8 Viewer Components
- `dataSource` + `srcField` props for API-based URL fetching
- `loading` state during fetch
- `refresh()` method
- Auto-detect response fields: `url`, `src`, `file_url`

#### Dialogs
- `size` prop (sm, md, lg, xl)
- `openDialog()`, `closeDialog()` methods
- ARIA `role="dialog"` / `role="alertdialog"`
- bc-dialog-wizard: `goToStep()`, `nextStep()`, `prevStep()`, `getCurrentStep()`
- bc-toast: `show(message?, variant?)`, `dismiss()`

#### Widgets
- `setValue()`, `getValue()` methods on all value-bearing widgets
- Mutable value props for programmatic updates

#### Layout
- bc-section: `toggle()`, `expand()`, `collapse()` methods
- bc-tabs: `selectTab()`, `getActiveIndex()` methods

#### Form Utilities
- `validateAllFields(container?)` — validate all fields, focus first invalid
- `resetAllFields(container?)` — reset all fields to defaults
- `clearAllErrors(container?)` — clear all validation errors
- `getFormData(container?)` — collect all field values as object

#### Reactivity
- `BcSetup.reactivity()` — register cross-field logic rules
- FormProxy: `getValue`, `setValue`, `setRequired`, `setReadonly`, `setDisabled`, `setError`, `clearError`, `setOptions`, `setVisible`
- Scoped to nearest form container

#### i18n
- English validation messages (default)
- Indonesian validation messages (`BcSetup.configure({ locale: 'id' })`)
- Extensible: add any locale

#### Testing
- 51 unit tests across 4 test suites (bc-setup, validation-engine, data-fetcher, field-utils)

#### Documentation
- 103 component doc files with props, events, methods tables
- 7 core guide docs (getting-started, bc-setup, theming, data-fetching, validation, reactivity, README)
- Component gallery/demo page at `/demo/`

### Changed
- All 103 components: `shadow: true` → `shadow: false` for easier theming
- `prefix` → `prefixText`, `suffix` → `suffixText` (HTMLElement reserved names)
- `focus()` → `focusField()`, `blur()` → `blurField()` (HTMLElement reserved names)

## [0.1.0] - 2026-04-27

### Initial Release
- 103 Stencil web components
- Fields, layout, views, charts, dialogs, widgets, viewers
- ECharts integration
- Tiptap rich text editor
- CodeMirror code editor
- Leaflet maps
- SignaturePad
- JsBarcode / QRCode
