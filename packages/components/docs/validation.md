# Validation — 3-Level Strategy

All field components support 3 levels of validation. Levels execute in order — stops at first failure.

## Level 1: Built-in Rules (via props)

Zero JavaScript. Set props on the component.

```html
<bc-field-string name="email" required min-length="5" max-length="100" pattern="^[^\s@]+@[^\s@]+$" />
<bc-field-integer name="age" required min="0" max="150" />
```

Built-in rules:

| Prop | Type | Description |
|------|------|-------------|
| `required` | boolean | Field must have a value |
| `min-length` | number | Minimum character count |
| `max-length` | number | Maximum character count |
| `min` | number | Minimum numeric value |
| `max` | number | Maximum numeric value |
| `pattern` | string | Regex pattern |
| `pattern-message` | string | Custom error for pattern failure |

## Level 2: Custom JS Validator (via JS property)

Set a function or array of validators on the element.

```javascript
const emailField = document.querySelector('[name="email"]');

emailField.customValidator = async (value) => {
  if (String(value).endsWith('@temp.com')) return 'Temporary emails not allowed';
  return null;  // null = valid
};
```

Or multiple validators:

```javascript
emailField.validators = [
  { rule: 'required', message: 'Email wajib diisi' },
  { rule: (value) => String(value).includes('@'), message: 'Must contain @' },
  { rule: 'no-competitor', message: 'Competitor emails blocked' }  // registered via BcSetup
];
```

Register named validators globally:

```javascript
BcSetup.registerValidator('no-competitor', async (value) => {
  if (String(value).endsWith('@competitor.com')) return 'Competitor emails not allowed';
  return null;
});
```

## Level 3: Server-side Validator (via JS property)

### URL-based

Component POSTs `{ value }` to the URL. Expects `{ valid: true }` or `{ valid: false, message: "..." }`.

```javascript
emailField.serverValidator = '/api/validate/email';
```

### Function-based

```javascript
emailField.serverValidator = async (value) => {
  const res = await fetch('/api/check-email', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: value })
  });
  const data = await res.json();
  if (data.exists) return 'Email already registered';
  return null;
};
```

## Execution Order

```
1. Built-in rules (sync)        → required, minLength, pattern, etc.
   ↓ all pass?
2. Custom validators (async)    → validators array + customValidator
   ↓ all pass?
3. Server-side validator (async) → serverValidator
   ↓ pass?
✓ VALID
```

If any level fails, subsequent levels are skipped.

## Validation Trigger

Control when validation runs via `validate-on` prop:

| Value | Behavior |
|-------|----------|
| `blur` (default) | Validate when field loses focus |
| `change` | Validate on every value change |
| `submit` | Validate only on form submit |
| `manual` | Validate only when `validate()` method is called |

```html
<bc-field-string name="username" required validate-on="change" />
```

Global default:

```javascript
BcSetup.configure({ validateOn: 'blur' });
```

## Validation State

Components expose validation state via props and methods:

| Prop | Values | Description |
|------|--------|-------------|
| `validation-status` | `none`, `validating`, `valid`, `invalid` | Current state |
| `validation-message` | string | Error or success message |

Methods:

```javascript
const result = await field.validate();  // { valid: boolean, errors: string[] }
field.setError('Custom error');
field.clearError();
```

## Custom Error Messages

Override default messages globally:

```javascript
BcSetup.configure({
  validationMessages: {
    required: 'Wajib diisi',
    minLength: 'Minimal {0} karakter',
    maxLength: 'Maksimal {0} karakter',
    email: 'Format email salah',
    pattern: 'Format tidak valid'
  }
});
```
