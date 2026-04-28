# Phase 6: Reactivity & Advanced — Design + Implementation (COMPLETED)

**Date:** 2026-07-28
**Status:** ✅ COMPLETE

## What was done

### BcSetup.reactivity() Runtime
- Document-level `lcFieldChange` listener auto-registered when `reactivity()` is called
- FormProxy scoped to nearest form container (`<form>`, `<bc-view-form>`, `[data-bc-form]`)
- Prevents cross-form interference with multiple forms on one page

### FormProxy API
- `getValue(name)` — get field value (calls component's `getValue()` method or reads `.value`)
- `setValue(name, value)` — set field value (calls component's `setValue()` method)
- `setRequired(name, bool)` — toggle required attribute
- `setReadonly(name, bool)` — toggle readonly attribute
- `setDisabled(name, bool)` — toggle disabled attribute
- `setError(name, message)` — set error (calls component's `setError()` method)
- `clearError(name)` — clear error (calls component's `clearError()` method)
- `setOptions(name, options)` — set select options (calls component's `setOptions()` method)
- `setVisible(name, bool)` — show/hide field via `display: none`

### Error Handling
- Reactivity handler errors caught and logged to console with field name context
- Does not break other reactivity rules or component behavior

### Usage
```javascript
BcSetup.reactivity({
  'type': (value, form) => {
    if (value === 'company') {
      form.setRequired('tax_id', true);
      form.setOptions('salutation', [{label:'PT',value:'pt'},{label:'CV',value:'cv'}]);
    }
  },
  'discount': (value, form) => {
    const price = form.getValue('price');
    const qty = form.getValue('quantity');
    form.setValue('total', Number(price) * Number(qty) * (1 - Number(value) / 100));
  }
});
```
