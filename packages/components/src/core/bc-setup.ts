import { BcConfig } from './types';

type ReactivityHandler = (value: unknown, form: FormProxy) => void;
type ValidatorFn = (value: unknown) => string | null | Promise<string | null>;

export interface FormProxy {
  getValue(name: string): unknown;
  setValue(name: string, value: unknown): void;
  setRequired(name: string, required: boolean): void;
  setReadonly(name: string, readonly: boolean): void;
  setDisabled(name: string, disabled: boolean): void;
  setError(name: string, message: string): void;
  clearError(name: string): void;
  setOptions(name: string, options: unknown[]): void;
  setVisible(name: string, visible: boolean): void;
}

const DEFAULT_CONFIG: BcConfig = {
  baseUrl: '',
  headers: {},
  auth: { type: 'none' },
  validateOn: 'blur',
  validationMessages: {
    required: 'This field is required',
    minLength: 'Minimum {0} characters',
    maxLength: 'Maximum {0} characters',
    min: 'Minimum value is {0}',
    max: 'Maximum value is {0}',
    pattern: 'Invalid format',
    email: 'Invalid email address',
    url: 'Invalid URL',
    phone: 'Invalid phone number',
  },
  size: 'md',
  locale: 'en',
  theme: 'light',
};

class BcSetupImpl {
  private _config: BcConfig = { ...DEFAULT_CONFIG, headers: {}, validationMessages: { ...DEFAULT_CONFIG.validationMessages } };
  private _reactivityRules: Map<string, ReactivityHandler> = new Map();
  private _validators: Map<string, ValidatorFn> = new Map();
  private _systemThemeCleanup: (() => void) | null = null;
  private _reactivityListener: ((e: Event) => void) | null = null;

  configure(partial: Partial<BcConfig>): void {
    if (partial.headers) {
      this._config.headers = { ...this._config.headers, ...partial.headers };
    }
    if (partial.auth) {
      this._config.auth = { ...this._config.auth, ...partial.auth };
    }
    if (partial.validationMessages) {
      this._config.validationMessages = { ...this._config.validationMessages, ...partial.validationMessages };
    }

    const keysToSkip = new Set(['headers', 'auth', 'validationMessages']);
    for (const key of Object.keys(partial) as Array<keyof BcConfig>) {
      if (!keysToSkip.has(key) && partial[key] !== undefined) {
        (this._config as unknown as Record<string, unknown>)[key] = partial[key];
      }
    }

    if (partial.theme !== undefined) {
      this._applyTheme(partial.theme);
    }
  }

  getConfig(): Readonly<BcConfig> {
    return this._config;
  }

  getHeaders(): Record<string, string> {
    const resolved: Record<string, string> = {};

    for (const [key, val] of Object.entries(this._config.headers)) {
      resolved[key] = typeof val === 'function' ? val() : val;
    }

    const auth = this._config.auth;
    if (auth.type === 'bearer') {
      const token = typeof auth.token === 'function' ? auth.token() : auth.token;
      if (token) {
        resolved['Authorization'] = `Bearer ${token}`;
      }
    } else if (auth.type === 'header' && auth.headerName) {
      const val = typeof auth.headerValue === 'function' ? auth.headerValue() : auth.headerValue;
      if (val) {
        resolved[auth.headerName] = val;
      }
    }

    return resolved;
  }

  getBaseUrl(): string {
    return this._config.baseUrl;
  }

  getValidationMessage(rule: string, ...params: unknown[]): string {
    let msg = this._config.validationMessages[rule] || rule;
    params.forEach((p, i) => {
      msg = msg.replace(`{${i}}`, String(p));
    });
    return msg;
  }

  reactivity(rules: Record<string, ReactivityHandler>): void {
    for (const [fieldName, handler] of Object.entries(rules)) {
      this._reactivityRules.set(fieldName, handler);
    }
    this._ensureReactivityListener();
  }

  private _ensureReactivityListener(): void {
    if (typeof document === 'undefined' || this._reactivityListener) return;
    this._reactivityListener = (e: Event) => {
      const detail = (e as CustomEvent).detail;
      if (!detail || !detail.name) return;
      const handler = this._reactivityRules.get(detail.name);
      if (!handler) return;
      const target = e.target as HTMLElement;
      if (!target) return;
      let container = target.parentElement;
      while (container) {
        if (container.tagName === 'FORM' || container.tagName === 'BC-VIEW-FORM' || container.hasAttribute('data-bc-form')) break;
        container = container.parentElement;
      }
      const scope = container || target.parentElement || document.body;
      const formProxy = this._createFormProxy(scope);
      try { handler(detail.value, formProxy); } catch (err) { console.error(`BcSetup reactivity error for field "${detail.name}":`, err); }
    };
    document.addEventListener('lcFieldChange', this._reactivityListener);
  }

  private _createFormProxy(container: HTMLElement): FormProxy {
    const getField = (name: string) => container.querySelector(`[name="${name}"]`) as (HTMLElement & Record<string, unknown>) | null;
    return {
      getValue(name: string): unknown { const f = getField(name); if (!f) return undefined; if (typeof f.getValue === 'function') return f.getValue(); return (f as unknown as HTMLInputElement).value; },
      setValue(name: string, value: unknown): void { const f = getField(name); if (!f) return; if (typeof f.setValue === 'function') f.setValue(value); else (f as unknown as HTMLInputElement).value = String(value ?? ''); },
      setRequired(name: string, required: boolean): void { const f = getField(name); if (!f) return; if (required) f.setAttribute('required', ''); else f.removeAttribute('required'); },
      setReadonly(name: string, readonly: boolean): void { const f = getField(name); if (!f) return; if (readonly) f.setAttribute('readonly', ''); else f.removeAttribute('readonly'); },
      setDisabled(name: string, disabled: boolean): void { const f = getField(name); if (!f) return; if (disabled) f.setAttribute('disabled', ''); else f.removeAttribute('disabled'); },
      setError(name: string, message: string): void { const f = getField(name); if (!f) return; if (typeof f.setError === 'function') f.setError(message); },
      clearError(name: string): void { const f = getField(name); if (!f) return; if (typeof f.clearError === 'function') f.clearError(); },
      setOptions(name: string, options: unknown[]): void { const f = getField(name); if (!f) return; if (typeof f.setOptions === 'function') f.setOptions(options); },
      setVisible(name: string, visible: boolean): void { const f = getField(name); if (!f) return; (f as HTMLElement).style.display = visible ? '' : 'none'; },
    };
  }

  getReactivityRule(fieldName: string): ReactivityHandler | undefined {
    return this._reactivityRules.get(fieldName);
  }

  hasReactivityRule(fieldName: string): boolean {
    return this._reactivityRules.has(fieldName);
  }

  registerValidator(name: string, fn: ValidatorFn): void {
    this._validators.set(name, fn);
  }

  getValidator(name: string): ValidatorFn | undefined {
    return this._validators.get(name);
  }

  reset(): void {
    this._cleanupSystemTheme();
    if (this._reactivityListener && typeof document !== 'undefined') {
      document.removeEventListener('lcFieldChange', this._reactivityListener);
      this._reactivityListener = null;
    }
    this._config = { ...DEFAULT_CONFIG, headers: {}, validationMessages: { ...DEFAULT_CONFIG.validationMessages } };
    this._reactivityRules.clear();
    this._validators.clear();
  }

  private _applyTheme(theme: string): void {
    if (typeof document === 'undefined') return;

    this._cleanupSystemTheme();

    if (theme === 'system') {
      const mq = window.matchMedia('(prefers-color-scheme: dark)');
      const apply = (e: MediaQueryList | MediaQueryListEvent) => {
        document.documentElement.setAttribute('data-bc-theme', e.matches ? 'dark' : 'light');
      };
      apply(mq);
      const handler = (e: MediaQueryListEvent) => apply(e);
      mq.addEventListener('change', handler);
      this._systemThemeCleanup = () => mq.removeEventListener('change', handler);
    } else if (theme === 'light') {
      document.documentElement.removeAttribute('data-bc-theme');
    } else {
      document.documentElement.setAttribute('data-bc-theme', theme);
    }
  }

  private _cleanupSystemTheme(): void {
    if (this._systemThemeCleanup) {
      this._systemThemeCleanup();
      this._systemThemeCleanup = null;
    }
  }
}

export const BcSetup = new BcSetupImpl();

if (typeof document !== 'undefined') {
  const baseUrlMeta = document.querySelector('meta[name="bc-base-url"]');
  const tokenMeta = document.querySelector('meta[name="bc-auth-token"]');
  const themeMeta = document.querySelector('meta[name="bc-theme"]');

  const autoConfig: Partial<BcConfig> = {};
  if (baseUrlMeta) {
    autoConfig.baseUrl = baseUrlMeta.getAttribute('content') || '';
  }
  if (tokenMeta) {
    autoConfig.auth = { type: 'bearer', token: tokenMeta.getAttribute('content') || '' };
  }
  if (themeMeta) {
    autoConfig.theme = (themeMeta.getAttribute('content') || 'light') as BcConfig['theme'];
  }
  if (Object.keys(autoConfig).length > 0) {
    BcSetup.configure(autoConfig);
  }
}
