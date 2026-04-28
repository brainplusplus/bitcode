# Phase 3: Select & Data-Driven Fields — Design + Implementation Plan

**Date:** 2026-07-28
**Depends on:** Phase 1 ✅, Phase 2 ✅
**Scope:** Wire 4-level data fetching to 5 select-family components

---

## WHAT CHANGES

### bc-field-select — MAJOR UPGRADE
Current: native `<select>` element, local options only.
Target: custom searchable dropdown, 4-level data, cascading.

New props:
- `searchable: boolean = false` — enable search/filter
- `serverSide: boolean = false` — fetch from API on search
- `multiple: boolean = false` — multi-select
- `displayField: string = 'label'` — which field to display
- `valueField: string = 'value'` — which field as value
- `groupBy: string = ''` — group options
- `creatable: boolean = false` — allow creating new options
- `pageSize: number = 50` — API page size
- `debounceMs: number = 300` — search debounce
- `minSearchLength: number = 1` — min chars before search
- `noResultsText: string = 'No results'`
- `loadingText: string = 'Loading...'`
- `fetchHeaders: string = ''` — custom headers JSON

New events:
- `lcBeforeFetch` — modify request before fetch
- `lcAfterFetch` — transform response after fetch
- `lcOptionsLoad` — options loaded
- `lcOptionsError` — load failed
- `lcOptionCreate` — new option created
- `lcDropdownOpen` — dropdown opened
- `lcDropdownClose` — dropdown closed

New methods:
- `loadOptions(query?)` — trigger fetch
- `reloadOptions()` — force reload
- `getOptions()` — get current options
- `getSelectedOptions()` — get selected
- `open()` — open dropdown
- `close()` — close dropdown

### bc-field-link — WIRE DATA FETCHER
Current: uses `getApiClient()` directly.
Target: use `fetchOptions()` from data-fetcher.ts, with `getApiClient()` as fallback.
Add `dataSource` prop support — if set, use URL instead of api-client.

### bc-field-dynlink — SAME AS LINK

### bc-field-tags — WIRE DATA FETCHER
Same pattern as link but multi-value.

### bc-field-tableselect — WIRE DATA FETCHER
Same pattern as tags.

---

## IMPLEMENTATION ORDER

1. Upgrade bc-field-select (biggest change — custom dropdown)
2. Wire bc-field-link to fetchOptions()
3. Wire bc-field-dynlink
4. Wire bc-field-tags
5. Wire bc-field-tableselect
6. Build & verify
7. Update 5 component docs
8. Commit & push

---

## KEY DESIGN: bc-field-select Custom Dropdown

The native `<select>` doesn't support search, custom rendering, or async loading.
Replace with custom div-based dropdown that:
- Shows search input when `searchable=true`
- Fetches from API when `serverSide=true` + `dataSource` set
- Falls back to local filter when `serverSide=false`
- Supports `{label, value}` object options
- Keyboard navigation (arrow keys, enter, escape)
- Click outside to close

---

**Generated: 2026-07-28**
