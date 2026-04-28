# bc-field-string

## Quick Start

```html
<bc-field-string name="myfield" label="My Field" />
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
| max | number | 0 | Max length |
| widget | string | '' | Embed widget (youtube, instagram, tiktok) |
| prefix-text | string | '' | Visual prefix |
| suffix-text | string | '' | Visual suffix |
| min-length | number | 0 | Min characters |
| max-length | number | 0 | Max characters |
| show-count | boolean | false | Character counter |
| pattern | string | '' | Regex validation |
| pattern-message | string | '' | Custom pattern error |
| depend-on | string | '' | Parent field for cascading |
| data-source | string | '' | API endpoint with {field} placeholders |

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


