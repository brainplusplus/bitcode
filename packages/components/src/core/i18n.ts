import { createStore } from '@stencil/store';

export type Locale = 'en' | 'id' | 'fr' | 'de' | 'es' | 'pt-BR' | 'ja' | 'zh-CN' | 'ko' | 'ar' | 'ru';
export type Translations = Record<string, string>;
export type Direction = 'ltr' | 'rtl';

const RTL_LOCALES: ReadonlySet<string> = new Set(['ar', 'he', 'fa', 'ur']);

const SUPPORTED_LOCALES: readonly Locale[] = [
  'en', 'id', 'fr', 'de', 'es', 'pt-BR', 'ja', 'zh-CN', 'ko', 'ar', 'ru',
] as const;

const { state, onChange } = createStore<{ locale: Locale }>({
  locale: 'en',
});

const registry = new Map<string, Translations>();

class I18n {

  readonly supportedLocales: readonly Locale[] = SUPPORTED_LOCALES;

  get locale(): Locale {
    return state.locale;
  }

  setLocale(locale: Locale): void {
    if (!SUPPORTED_LOCALES.includes(locale)) {
      console.warn(`[i18n] Unsupported locale "${locale}". Falling back to "en".`);
      state.locale = 'en';
      return;
    }
    state.locale = locale;
  }

  /** Subscribe to locale changes. Returns unsubscribe function. */
  onLocaleChange(callback: (locale: Locale) => void): () => void {
    return onChange('locale', callback);
  }

  get dir(): Direction {
    return RTL_LOCALES.has(state.locale) ? 'rtl' : 'ltr';
  }

  get isRTL(): boolean {
    return RTL_LOCALES.has(state.locale);
  }

  /**
   * Register translations for a locale. Merges with existing.
   * Can be called multiple times for app-level overrides.
   */
  registerTranslations(locale: string, translations: Translations): void {
    const existing = registry.get(locale) || {};
    registry.set(locale, { ...existing, ...translations });
  }

  /** Translate key with interpolation. Fallback: current locale -> en -> key. */
  t(key: string, params?: Record<string, string | number>): string {
    const currentDict = registry.get(state.locale);
    const fallbackDict = registry.get('en');

    let text = currentDict?.[key] ?? fallbackDict?.[key] ?? key;

    if (params) {
      for (const [k, v] of Object.entries(params)) {
        text = text.replace(new RegExp(`\\{${k}\\}`, 'g'), String(v));
      }
    }

    return text;
  }

  /** Pluralize using Intl.PluralRules. Looks up key_{zero|one|two|few|many|other}. */
  plural(key: string, count: number, params?: Record<string, string | number>): string {
    const rule = new Intl.PluralRules(state.locale).select(count);
    const mergedParams: Record<string, string | number> = { count, ...params };

    const pluralKey = `${key}_${rule}`;
    const currentDict = registry.get(state.locale);
    const fallbackDict = registry.get('en');

    if (currentDict?.[pluralKey] !== undefined || fallbackDict?.[pluralKey] !== undefined) {
      return this.t(pluralKey, mergedParams);
    }

    const otherKey = `${key}_other`;
    if (currentDict?.[otherKey] !== undefined || fallbackDict?.[otherKey] !== undefined) {
      return this.t(otherKey, mergedParams);
    }

    return this.t(key, mergedParams);
  }

  readonly tf = {
    /** Format date using Intl.DateTimeFormat with current locale. */
    date: (value: Date | string | number, options?: Intl.DateTimeFormatOptions): string => {
      try {
        const date = value instanceof Date ? value : new Date(value);
        return new Intl.DateTimeFormat(state.locale, options).format(date);
      } catch {
        return String(value);
      }
    },

    /** Format number using Intl.NumberFormat with current locale. */
    number: (value: number, options?: Intl.NumberFormatOptions): string => {
      try {
        return new Intl.NumberFormat(state.locale, options).format(value);
      } catch {
        return String(value);
      }
    },

    /** Format currency. Requires currency code (e.g., 'USD', 'IDR'). */
    currency: (value: number, currency: string, options?: Intl.NumberFormatOptions): string => {
      try {
        return new Intl.NumberFormat(state.locale, {
          style: 'currency',
          currency,
          ...options,
        }).format(value);
      } catch {
        return String(value);
      }
    },

    /** Format relative time, e.g. tf.relativeTime(-1, 'day') -> "yesterday". */
    relativeTime: (value: number, unit: Intl.RelativeTimeFormatUnit): string => {
      try {
        return new Intl.RelativeTimeFormat(state.locale, { numeric: 'auto' }).format(value, unit);
      } catch {
        return String(value);
      }
    },
  };
}

export const i18n = new I18n();
