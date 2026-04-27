# bc-field-image

## Quick Start

```html
<bc-field-image name="myfield" label="My Field" />
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

