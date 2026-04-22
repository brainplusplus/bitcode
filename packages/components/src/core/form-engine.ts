import { evaluate } from '../utils/expression-eval';

interface FieldConfig {
  type: string;
  depends_on?: string;
  readonly_if?: string;
  mandatory_if?: string;
  fetch_from?: string;
  formula?: string;
  required?: boolean;
  readonly?: boolean;
  default?: unknown;
}

interface FormConfig {
  fields: Record<string, FieldConfig>;
}

type ChangeCallback = (field: string, value: unknown, computed: Record<string, unknown>) => void;

export class FormEngine {
  private config: FormConfig;
  private values: Record<string, unknown> = {};
  private listeners: Set<ChangeCallback> = new Set();

  constructor(config: FormConfig) {
    this.config = config;
  }

  setValues(values: Record<string, unknown>): void {
    this.values = { ...this.values, ...values };
    this.recompute();
  }

  setValue(field: string, value: unknown): void {
    const oldValue = this.values[field];
    this.values[field] = value;
    this.recompute();
    this.notify(field, value, oldValue);
  }

  getValue(field: string): unknown {
    const cfg = this.config.fields[field];
    if (cfg?.formula) {
      return this.getComputedValue(field);
    }
    return this.values[field];
  }

  getValues(): Record<string, unknown> {
    const result: Record<string, unknown> = { ...this.values };
    for (const [name, cfg] of Object.entries(this.config.fields)) {
      if (cfg.formula) {
        result[name] = this.getComputedValue(name);
      }
    }
    return result;
  }

  isVisible(field: string): boolean {
    const cfg = this.config.fields[field];
    if (!cfg?.depends_on) return true;
    return !!evaluate(cfg.depends_on, this.values);
  }

  isReadonly(field: string): boolean {
    const cfg = this.config.fields[field];
    if (cfg?.readonly) return true;
    if (!cfg?.readonly_if) return false;
    return !!evaluate(cfg.readonly_if, this.values);
  }

  isMandatory(field: string): boolean {
    const cfg = this.config.fields[field];
    if (cfg?.required) return true;
    if (!cfg?.mandatory_if) return false;
    return !!evaluate(cfg.mandatory_if, this.values);
  }

  getComputedValue(field: string): unknown {
    const cfg = this.config.fields[field];
    if (!cfg?.formula) return this.values[field];
    return evaluate(cfg.formula, this.values);
  }

  getFetchPath(field: string): string | undefined {
    return this.config.fields[field]?.fetch_from;
  }

  getFieldsToFetch(): Array<{ field: string; path: string }> {
    const result: Array<{ field: string; path: string }> = [];
    for (const [name, cfg] of Object.entries(this.config.fields)) {
      if (cfg.fetch_from) {
        result.push({ field: name, path: cfg.fetch_from });
      }
    }
    return result;
  }

  getDefaults(): Record<string, unknown> {
    const defaults: Record<string, unknown> = {};
    for (const [name, cfg] of Object.entries(this.config.fields)) {
      if (cfg.default !== undefined) {
        defaults[name] = cfg.default;
      }
    }
    return defaults;
  }

  validate(): Array<{ field: string; message: string }> {
    const errors: Array<{ field: string; message: string }> = [];
    for (const name of Object.keys(this.config.fields)) {
      if (!this.isVisible(name)) continue;
      if (this.isMandatory(name)) {
        const val = this.getValue(name);
        if (val === null || val === undefined || val === '') {
          errors.push({ field: name, message: `${name} is required` });
        }
      }
    }
    return errors;
  }

  onChange(callback: ChangeCallback): () => void {
    this.listeners.add(callback);
    return () => this.listeners.delete(callback);
  }

  private recompute(): void {
    // Recompute all formula fields
    for (const [name, cfg] of Object.entries(this.config.fields)) {
      if (cfg.formula) {
        const computed = evaluate(cfg.formula, this.values);
        if (computed !== undefined) {
          this.values[name] = computed;
        }
      }
    }
  }

  private notify(field: string, _value: unknown, _oldValue: unknown): void {
    const computed: Record<string, unknown> = {};
    for (const [name, cfg] of Object.entries(this.config.fields)) {
      if (cfg.formula) {
        computed[name] = this.values[name];
      }
    }
    this.listeners.forEach(cb => {
      try { cb(field, this.values[field], computed); } catch { /* swallow */ }
    });
  }
}
