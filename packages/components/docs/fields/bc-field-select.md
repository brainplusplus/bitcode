# bc-field-select

## Quick Start

```html
<bc-field-select name="myfield" label="My Field" />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| name | string | '' | Field identifier |
| label | string | '' | Display label |
| value | varies | | Field value |
| required | boolean | false | Required |
| readonly | boolean | false | Read-only |
| disabled | boolean | false | Disabled |
| validation-status | string | 'none' | Validation state |
| validation-message | string | '' | Error message |
| hint | string | '' | Helper text |
| size | string | 'md' | sm/md/lg |
| clearable | boolean | false | Clear button |
| tooltip | string | '' | Tooltip |
| loading | boolean | false | Loading state |
| autofocus | boolean | false | Auto focus |
| default-value | varies | | Default for reset |
| validate-on | string | 'blur' | Validation trigger |


### Component-Specific Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| searchable | boolean | false | Enable search |
| server-side | boolean | false | Server-side search |
| multiple | boolean | false | Multi-select |
| display-field | string | 'label' | Display field name |
| value-field | string | 'value' | Value field name |
| group-by | string | '' | Group options by field |
| creatable | boolean | false | Allow creating options |
| page-size | number | 50 | API page size |
| debounce-ms | number | 300 | Search debounce |
| min-search-length | number | 1 | Min chars before search |
| no-results-text | string | 'No results' | Empty state text |
| depend-on | string | '' | Parent field for cascading |
| data-source | string | '' | API endpoint |
| fetch-headers | string | '' | Custom headers JSON |
| model | string | '' | BitCode model name |

## Events

| Event | Payload |
|-------|---------|
| lcFieldChange | {name, value, oldValue} |
| lcFieldFocus | {name, value} |
| lcFieldBlur | {name, value, dirty, touched} |
| lcFieldClear | {name, oldValue} |
| lcFieldInvalid | {name, value, errors} |
| lcFieldValid | {name, value} |

## Methods

| Method | Returns |
|--------|---------|
| validate() | Promise<{valid, errors}> |
| reset() | Promise<void> |
| clear() | Promise<void> |
| setValue(value, emit?) | Promise<void> |
| getValue() | Promise<value> |
| focusField() | Promise<void> |
| blurField() | Promise<void> |
| isDirty() | Promise<boolean> |
| isTouched() | Promise<boolean> |
| setError(msg) | Promise<void> |
| clearError() | Promise<void> |

See [validation](../validation.md), [theming](../theming.md), [data-fetching](../data-fetching.md).


