# Phase 2: Field Components — Implementation Plan

**Date:** 2026-07-27
**Depends on:** Phase 1 (core infrastructure) ✅
**Scope:** Upgrade all 34 field components with enterprise props/events/methods
**Approach:** Reference implementation first (bc-field-string), then batch by pattern group

---

## ANALYSIS: 6 Pattern Groups

### Group A — Text Input (10 components)
bc-field-string, bc-field-password, bc-field-date, bc-field-datetime, bc-field-time, bc-field-integer, bc-field-float, bc-field-decimal, bc-field-currency, bc-field-percent

Has `<input>` element. Focus/blur attach to input. Prefix/suffix wrap input. Clearable shows X button. minLength/maxLength/showCount applicable for string/password.

### Group B — Textarea (2 components)
bc-field-text, bc-field-smalltext

Has `<textarea>`. Same as Group A but multi-line. showCount applicable.

### Group C — Boolean/Choice (10 components)
bc-field-checkbox, bc-field-toggle, bc-field-radio, bc-field-multicheck, bc-field-rating, bc-field-color, bc-field-barcode, bc-field-duration, bc-field-geo, bc-field-signature

No text input. Focus/blur on wrapper or specific element. No prefix/suffix. No minLength/maxLength. Clearable = reset to default. Some missing required/readonly.

### Group D — Rich Editor (5 components)
bc-field-richtext, bc-field-markdown, bc-field-html, bc-field-code, bc-field-json

3rd party editor (Tiptap, CodeMirror). Focus/blur via editor API. showCount on content length. shadow: false already.

### Group E — Data-driven (5 components)
bc-field-select, bc-field-link, bc-field-dynlink, bc-field-tags, bc-field-tableselect

Needs 4-level data fetching. Phase 3 handles data features. Phase 2 only adds universal props/events/methods.

### Group F — File Upload (2 components)
bc-field-file, bc-field-image

Already feature-rich. Phase 2 adds universal props/events/methods only.

---

## WHAT CHANGES PER COMPONENT

### Universal additions (ALL 34):

**Props:** validationStatus, validationMessage, hint, size, clearable, tooltip, loading, autofocus, defaultValue, validateOn, dependOn, dataSource
Plus prefix/suffix for Group A/B. Plus minLength/maxLength/showCount/pattern/patternMessage for text-based. Plus missing required/readonly where absent.

**Events:** lcFieldFocus, lcFieldBlur, lcFieldClear, lcFieldInvalid, lcFieldValid

**Methods:** validate(), reset(), clear(), setValue(), getValue(), focus(), blur(), isDirty(), isTouched(), setError(), clearError()

**Internal:** _dirty, _touched, _errors, _initialValue

**Shadow DOM:** shadow: true → shadow: false (21 components)

---

## IMPLEMENTATION ORDER

| Step | What | Components |
|------|------|-----------|
| 1 | Reference implementation | bc-field-string |
| 2 | Update field-base.css | Size variants, validation states, hint, counter, prefix/suffix, clearable, tooltip, loading |
| 3 | Build & verify reference | npm run build |
| 4 | Group A batch | 9 remaining text inputs |
| 5 | Group B | 2 textareas |
| 6 | Group C | 10 boolean/choice |
| 7 | Group D | 5 rich editors |
| 8 | Group E | 5 data-driven (universal only) |
| 9 | Group F | 2 file upload (universal only) |
| 10 | Build & verify all | npm run build |
| 11 | Generate docs | 34 component doc files |
| 12 | Update project docs | AGENTS.md, codebase.md, features.md |
| 13 | Commit & push | |

---

## SPECIFIC GAPS PER COMPONENT

| Component | Missing Props | Missing Events/Methods | Shadow |
|-----------|--------------|----------------------|--------|
| bc-field-checkbox | required, readonly | all universal | true→false |
| bc-field-toggle | required, readonly | all universal | true→false |
| bc-field-color | required, readonly | all universal | true→false |
| bc-field-rating | required, readonly | all universal | true→false |
| bc-field-radio | required, readonly | all universal | true→false |
| bc-field-barcode | required, readonly, placeholder | all universal | false (keep) |
| bc-field-duration | required, readonly, placeholder | all universal | true→false |
| bc-field-geo | required, readonly, placeholder | all universal | false (keep) |
| bc-field-signature | required, readonly | all universal | false (keep) |
| bc-field-float | min, max, step | all universal | true→false |
| bc-field-decimal | min, max, step | all universal | true→false |
| bc-field-currency | min, max, step | all universal | true→false |
| All others | (have basic props) | all universal | varies |

---

## RISK: `:host` with shadow: false

Stencil compiles `:host` to `[tag-name]` selector when shadow: false. Verified in Stencil docs. No CSS breakage.

---

**Generated: 2026-07-27**
