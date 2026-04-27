# Phase 2: Field Components — Design + Implementation Plan

**Date:** 2026-07-27
**Depends on:** Phase 1 (core infrastructure)
**Scope:** Upgrade all 34 field components with universal props/events/methods
**Ref:** See `enterprise-upgrade-master.md` Section 2-4 for full prop/event/method specs

---

## SCOPE

Apply universal enterprise features to all 34 `bc-field-*` components:
- Add missing standard props (required, readonly, disabled where absent)
- Add enterprise props (validationStatus, hint, size, clearable, prefix, suffix, tooltip, loading, autofocus, defaultValue, validateOn)
- Add text-specific props where applicable (minLength, maxLength, showCount)
- Add numeric-specific props where applicable (min, max, step)
- Add universal events (lcFieldFocus, lcFieldBlur, lcFieldClear, lcFieldInvalid, lcFieldValid)
- Add universal methods (validate, reset, clear, setValue, getValue, focus, blur, isDirty, isTouched, setError, clearError)
- Wire to field-utils.ts for validation, dirty/touched tracking, ARIA
- Generate `docs/components/fields/bc-field-*.md` for each component

---

## COMPONENT LIST (34)

### Group A: Simple Text Input (pattern identical, batch-update)
bc-field-string, bc-field-password, bc-field-date, bc-field-datetime, bc-field-time

### Group B: Textarea
bc-field-text, bc-field-smalltext

### Group C: Numeric
bc-field-integer, bc-field-float, bc-field-decimal, bc-field-currency, bc-field-percent

### Group D: Rich Editor (CodeMirror/TipTap)
bc-field-code, bc-field-json, bc-field-richtext, bc-field-markdown, bc-field-html

### Group E: Boolean/Choice (no text input)
bc-field-checkbox, bc-field-toggle, bc-field-radio, bc-field-multicheck

### Group F: Special
bc-field-color, bc-field-rating, bc-field-barcode, bc-field-duration, bc-field-geo, bc-field-signature

### Group G: Data-driven (upgraded further in Phase 3)
bc-field-select, bc-field-link, bc-field-dynlink, bc-field-tags, bc-field-tableselect

### Group H: File/Upload
bc-field-file, bc-field-image

---

## IMPLEMENTATION ORDER

1. Pick one component from Group A (bc-field-string) as **reference implementation**
2. Verify it works: build, test manually
3. Apply same pattern to rest of Group A (4 components)
4. Group B (2), Group C (5), Group D (5), Group E (4), Group F (6)
5. Group G — universal props only (data-driven features in Phase 3)
6. Group H — universal props only (upload features already good)
7. Generate docs: `docs/components/fields/bc-field-*.md` (34 files)

---

## PER-COMPONENT GAP TABLE

See `enterprise-upgrade-master.md` Section 11 for the full matrix of what each component needs.

Key gaps to fix:
- bc-field-checkbox: missing `required`
- bc-field-toggle: missing `required`, `readonly`
- bc-field-color: missing `required`, `readonly`
- bc-field-rating: missing `required`, `readonly`
- bc-field-radio: missing `required`, `readonly`
- bc-field-barcode: missing `required`, `readonly`, `placeholder`
- bc-field-duration: missing `required`, `readonly`, `placeholder`
- bc-field-geo: missing `required`, `readonly`, `placeholder`
- bc-field-signature: missing `required`, `readonly`
- bc-field-float/decimal/currency: missing `min`, `max`, `step`

---

**Generated: 2026-07-27**
