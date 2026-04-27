import { ValidationResult } from './types';
import { BuiltInValidationOpts, runValidationPipeline } from './validation-engine';

export interface FieldState {
  dirty: boolean;
  touched: boolean;
  pristine: boolean;
  initialValue: unknown;
}

export function createFieldState(initialValue: unknown): FieldState {
  return {
    dirty: false,
    touched: false,
    pristine: true,
    initialValue,
  };
}

export function markDirty(state: FieldState, currentValue: unknown): FieldState {
  const dirty = currentValue !== state.initialValue;
  return { ...state, dirty, pristine: !dirty };
}

export function markTouched(state: FieldState): FieldState {
  return { ...state, touched: true };
}

export function resetFieldState(state: FieldState, newInitialValue?: unknown): FieldState {
  return createFieldState(newInitialValue !== undefined ? newInitialValue : state.initialValue);
}

export function getAriaAttrs(props: {
  name?: string;
  required?: boolean;
  disabled?: boolean;
  readonly?: boolean;
  validationStatus?: string;
  validationMessage?: string;
  hint?: string;
}): Record<string, string> {
  const attrs: Record<string, string> = {};

  if (props.required) attrs['aria-required'] = 'true';
  if (props.disabled) attrs['aria-disabled'] = 'true';
  if (props.readonly) attrs['aria-readonly'] = 'true';

  if (props.validationStatus === 'invalid') {
    attrs['aria-invalid'] = 'true';
  }

  if (props.validationMessage) {
    const errorId = `${props.name || 'field'}-error`;
    attrs['aria-errormessage'] = errorId;
  }

  if (props.hint) {
    const hintId = `${props.name || 'field'}-hint`;
    attrs['aria-describedby'] = hintId;
  }

  return attrs;
}

export function getFieldClasses(opts: {
  size?: string;
  validationStatus?: string;
  disabled?: boolean;
  readonly?: boolean;
  loading?: boolean;
  dirty?: boolean;
  touched?: boolean;
}): Record<string, boolean> {
  return {
    'bc-field': true,
    'bc-field-sm': opts.size === 'sm',
    'bc-field-md': opts.size === 'md' || !opts.size,
    'bc-field-lg': opts.size === 'lg',
    'bc-field-valid': opts.validationStatus === 'valid',
    'bc-field-invalid': opts.validationStatus === 'invalid',
    'bc-field-validating': opts.validationStatus === 'validating',
    'bc-field-disabled': !!opts.disabled,
    'bc-field-readonly': !!opts.readonly,
    'bc-field-loading': !!opts.loading,
    'bc-field-dirty': !!opts.dirty,
    'bc-field-touched': !!opts.touched,
  };
}

export function getInputClasses(opts: {
  size?: string;
  validationStatus?: string;
}): Record<string, boolean> {
  return {
    'bc-field-input': true,
    'bc-field-input-sm': opts.size === 'sm',
    'bc-field-input-lg': opts.size === 'lg',
    'error': opts.validationStatus === 'invalid',
    'valid': opts.validationStatus === 'valid',
  };
}

export async function validateFieldValue(
  value: unknown,
  builtIn: BuiltInValidationOpts,
  extra?: {
    validators?: import('./types').ValidationRule[];
    customValidator?: (value: unknown) => string | null | Promise<string | null>;
    serverValidator?: string | ((value: unknown) => Promise<string | null>);
    serverValidatorHeaders?: Record<string, string>;
  },
): Promise<ValidationResult> {
  return runValidationPipeline({
    value,
    builtIn,
    validators: extra?.validators,
    customValidator: extra?.customValidator,
    serverValidator: extra?.serverValidator,
    serverValidatorHeaders: extra?.serverValidatorHeaders,
  });
}

let debounceTimers: Map<string, ReturnType<typeof setTimeout>> = new Map();

export function debounce(key: string, fn: () => void, ms: number): void {
  const existing = debounceTimers.get(key);
  if (existing) clearTimeout(existing);
  debounceTimers.set(key, setTimeout(() => {
    debounceTimers.delete(key);
    fn();
  }, ms));
}

export function findFormContainer(element: HTMLElement): HTMLElement {
  let parent = element.parentElement;
  while (parent) {
    if (parent.tagName === 'FORM' ||
        parent.tagName === 'BC-VIEW-FORM' ||
        parent.hasAttribute('data-bc-form')) {
      return parent;
    }
    parent = parent.parentElement;
  }
  return element.parentElement || document.body;
}

export function findSiblingField(container: HTMLElement, fieldName: string): HTMLElement | null {
  return container.querySelector(`[name="${fieldName}"]`);
}

export function createFormProxy(container: HTMLElement): import('./bc-setup').FormProxy {
  const getField = (name: string) => findSiblingField(container, name) as (HTMLElement & Record<string, unknown>) | null;

  return {
    getValue(name: string): unknown {
      const field = getField(name);
      if (!field) return undefined;
      if (typeof field.getValue === 'function') return field.getValue();
      return (field as unknown as HTMLInputElement).value;
    },

    setValue(name: string, value: unknown): void {
      const field = getField(name);
      if (!field) return;
      if (typeof field.setValue === 'function') {
        field.setValue(value);
      } else {
        (field as unknown as HTMLInputElement).value = String(value ?? '');
      }
    },

    setRequired(name: string, required: boolean): void {
      const field = getField(name);
      if (!field) return;
      if (required) field.setAttribute('required', '');
      else field.removeAttribute('required');
    },

    setReadonly(name: string, readonly: boolean): void {
      const field = getField(name);
      if (!field) return;
      if (readonly) field.setAttribute('readonly', '');
      else field.removeAttribute('readonly');
    },

    setDisabled(name: string, disabled: boolean): void {
      const field = getField(name);
      if (!field) return;
      if (disabled) field.setAttribute('disabled', '');
      else field.removeAttribute('disabled');
    },

    setError(name: string, message: string): void {
      const field = getField(name);
      if (!field) return;
      if (typeof field.setError === 'function') field.setError(message);
    },

    clearError(name: string): void {
      const field = getField(name);
      if (!field) return;
      if (typeof field.clearError === 'function') field.clearError();
    },

    setOptions(name: string, options: unknown[]): void {
      const field = getField(name);
      if (!field) return;
      if (typeof field.setOptions === 'function') field.setOptions(options);
    },

    setVisible(name: string, visible: boolean): void {
      const field = getField(name);
      if (!field) return;
      (field as HTMLElement).style.display = visible ? '' : 'none';
    },
  };
}
