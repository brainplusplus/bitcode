# Reactivity — Dependent Fields & Cross-Field Logic

Two approaches for field reactivity. Use either or both.

## Approach 1: Declarative (via props)

For cascading data — parent field changes, child field reloads options.

```html
<bc-field-select name="province" options='[{"label":"Jawa Barat","value":"jabar"}]' />
<bc-field-select name="city" depend-on="province" data-source="/api/cities?province={province}" />
<bc-field-select name="district" depend-on="city" data-source="/api/districts?city={city}" />
```

When province changes:
1. City detects change via `lcFieldChange` event
2. City replaces `{province}` in its `data-source` URL with the new value
3. City fetches new options
4. City clears its current value
5. City emits its own `lcFieldChange`, cascading to district

Multiple parents: `depend-on="province,type"` (comma-separated).

## Approach 2: Imperative (via BcSetup.reactivity)

For complex logic — set values, toggle required, conditional options, cross-field validation.

```javascript
BcSetup.reactivity({
  'type': (value, form) => {
    if (value === 'company') {
      form.setRequired('tax_id', true);
      form.setValue('is_company', true);
      form.setOptions('salutation', [
        { label: 'PT', value: 'pt' },
        { label: 'CV', value: 'cv' }
      ]);
    } else {
      form.setRequired('tax_id', false);
      form.setValue('is_company', false);
      form.setOptions('salutation', [
        { label: 'Mr', value: 'mr' },
        { label: 'Mrs', value: 'mrs' }
      ]);
    }
  },

  'discount': (value, form) => {
    const price = form.getValue('price');
    const qty = form.getValue('quantity');
    form.setValue('total', Number(price) * Number(qty) * (1 - Number(value) / 100));
  },

  'end_date': (value, form) => {
    const start = form.getValue('start_date');
    if (start && value && new Date(String(value)) < new Date(String(start))) {
      form.setError('end_date', 'End date must be after start date');
    } else {
      form.clearError('end_date');
    }
  }
});
```

## FormProxy API

The `form` parameter in reactivity callbacks provides:

| Method | Description |
|--------|-------------|
| `getValue(name)` | Get field value |
| `setValue(name, value)` | Set field value |
| `setRequired(name, bool)` | Toggle required |
| `setReadonly(name, bool)` | Toggle readonly |
| `setDisabled(name, bool)` | Toggle disabled |
| `setError(name, message)` | Set error message |
| `clearError(name)` | Clear error |
| `setOptions(name, options)` | Set select options |
| `setVisible(name, bool)` | Show/hide field |

FormProxy automatically scopes to the nearest form container — safe with multiple forms on one page.

## Scoping

Reactivity rules are scoped to the nearest form container:
- `<form>` element
- `<bc-view-form>` component
- Element with `data-bc-form` attribute

This prevents cross-form interference when multiple forms exist on one page.

## Execution Flow

```
Field emits lcFieldChange
  ↓
1. Declarative: sibling with depend-on="thisField" re-fetches data
  ↓
2. Imperative: BcSetup.reactivity rule for this field executes
  ↓
3. FormEngine (if present): depends_on, readonly_if, mandatory_if, formula
```

All three can coexist. Declarative and imperative are standalone. FormEngine is BitCode-specific.
