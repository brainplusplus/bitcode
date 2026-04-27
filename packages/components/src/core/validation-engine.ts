import { ValidationResult, ValidationRule } from './types';
import { BcSetup } from './bc-setup';
import {
  validateRequired,
  validateMinLength,
  validateMaxLength,
  validateMin,
  validateMax,
  validatePattern,
  validateEmail,
  validateUrl,
  validatePhone,
} from '../utils/validators';

export interface BuiltInValidationOpts {
  required?: boolean;
  minLength?: number;
  maxLength?: number;
  min?: number;
  max?: number;
  pattern?: string;
  patternMessage?: string;
  type?: string;
}

export function validateBuiltIn(value: unknown, opts: BuiltInValidationOpts): ValidationResult {
  const errors: string[] = [];
  const strVal = value === null || value === undefined ? '' : String(value);
  const isEmpty = !validateRequired(value);

  if (opts.required && isEmpty) {
    errors.push(BcSetup.getValidationMessage('required'));
    return { valid: false, errors };
  }

  if (isEmpty) {
    return { valid: true, errors: [] };
  }

  if (opts.type === 'email' && !validateEmail(strVal)) {
    errors.push(BcSetup.getValidationMessage('email'));
  }
  if (opts.type === 'url' && !validateUrl(strVal)) {
    errors.push(BcSetup.getValidationMessage('url'));
  }
  if (opts.type === 'phone' && !validatePhone(strVal)) {
    errors.push(BcSetup.getValidationMessage('phone'));
  }

  if (opts.minLength && opts.minLength > 0 && !validateMinLength(strVal, opts.minLength)) {
    errors.push(BcSetup.getValidationMessage('minLength', opts.minLength));
  }
  if (opts.maxLength && opts.maxLength > 0 && !validateMaxLength(strVal, opts.maxLength)) {
    errors.push(BcSetup.getValidationMessage('maxLength', opts.maxLength));
  }

  if (opts.min !== undefined && typeof value === 'number' && !validateMin(value, opts.min)) {
    errors.push(BcSetup.getValidationMessage('min', opts.min));
  }
  if (opts.max !== undefined && typeof value === 'number' && !validateMax(value, opts.max)) {
    errors.push(BcSetup.getValidationMessage('max', opts.max));
  }

  if (opts.pattern && !validatePattern(strVal, opts.pattern)) {
    errors.push(opts.patternMessage || BcSetup.getValidationMessage('pattern'));
  }

  return { valid: errors.length === 0, errors };
}

export async function validateCustom(value: unknown, validators: ValidationRule[]): Promise<ValidationResult> {
  const errors: string[] = [];

  for (const v of validators) {
    if (typeof v.rule === 'string') {
      const registeredFn = BcSetup.getValidator(v.rule);
      if (registeredFn) {
        const result = await registeredFn(value);
        if (result) {
          errors.push(result);
        }
      }
      continue;
    }

    if (typeof v.rule === 'function') {
      const result = await v.rule(value);
      if (!result) {
        errors.push(v.message);
      }
    }
  }

  return { valid: errors.length === 0, errors };
}

export async function validateServer(
  value: unknown,
  validator: string | ((value: unknown) => Promise<string | null>),
  headers?: Record<string, string>,
): Promise<ValidationResult> {
  if (typeof validator === 'function') {
    const error = await validator(value);
    return error ? { valid: false, errors: [error] } : { valid: true, errors: [] };
  }

  const allHeaders = {
    'Content-Type': 'application/json',
    ...BcSetup.getHeaders(),
    ...(headers || {}),
  };

  const baseUrl = BcSetup.getBaseUrl();
  let url = validator;
  if (url && !url.startsWith('http') && baseUrl) {
    url = baseUrl + url;
  }

  const res = await fetch(url, {
    method: 'POST',
    headers: allHeaders,
    body: JSON.stringify({ value }),
  });

  if (!res.ok) {
    return { valid: false, errors: [`Validation server error: ${res.status}`] };
  }

  const data = await res.json();

  if (typeof data === 'object' && data !== null) {
    const obj = data as Record<string, unknown>;
    if (obj.valid === false || obj.error || obj.message) {
      const msg = String(obj.error || obj.message || 'Validation failed');
      return { valid: false, errors: [msg] };
    }
  }

  return { valid: true, errors: [] };
}

export async function runValidationPipeline(opts: {
  value: unknown;
  builtIn?: BuiltInValidationOpts;
  validators?: ValidationRule[];
  customValidator?: (value: unknown) => string | null | Promise<string | null>;
  serverValidator?: string | ((value: unknown) => Promise<string | null>);
  serverValidatorHeaders?: Record<string, string>;
}): Promise<ValidationResult> {
  if (opts.builtIn) {
    const builtInResult = validateBuiltIn(opts.value, opts.builtIn);
    if (!builtInResult.valid) return builtInResult;
  }

  if (opts.validators && opts.validators.length > 0) {
    const customResult = await validateCustom(opts.value, opts.validators);
    if (!customResult.valid) return customResult;
  }

  if (opts.customValidator) {
    const error = await opts.customValidator(opts.value);
    if (error) return { valid: false, errors: [error] };
  }

  if (opts.serverValidator) {
    const serverResult = await validateServer(opts.value, opts.serverValidator, opts.serverValidatorHeaders);
    if (!serverResult.valid) return serverResult;
  }

  return { valid: true, errors: [] };
}
