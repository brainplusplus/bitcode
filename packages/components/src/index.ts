export { BcSetup } from './core/bc-setup';
export { validateAllFields, resetAllFields, clearAllErrors, getFormData } from './core/bc-setup';

export type {
  BcConfig,
  BcAuthConfig,
  ValidateOn,
  ValidationResult,
  ValidationRule,
  FetchParams,
  FetchResult,
  DataFetcher,
  OptionsFetcher,
  FieldChangeEvent,
  FieldFocusEvent,
  FieldBlurEvent,
  FieldClearEvent,
  FieldValidationEvent,
  FieldValidEvent,
  BeforeFetchEvent,
  AfterFetchEvent,
  OptionsLoadEvent,
  OptionsErrorEvent,
  OptionCreateEvent,
  ChartClickEvent,
  ChartHoverEvent,
  RowEditEvent,
  RowExpandEvent,
  CellClickEvent,
  ColumnResizeEvent,
  PageChangeEvent,
  SortChangeEvent,
  FilterChangeEvent,
  FormSubmitEvent,
  FieldType,
  WidgetType,
  FieldConfig,
  FieldBehavior,
} from './core/types';

export { fetchData, fetchOptions, resolveUrl, normalizeResponse, buildHeaders } from './core/data-fetcher';
export { validateBuiltIn, runValidationPipeline, validateServer, validateCustom } from './core/validation-engine';
export type { BuiltInValidationOpts } from './core/validation-engine';

export {
  createFieldState,
  markDirty,
  markTouched,
  resetFieldState,
  getAriaAttrs,
  getFieldClasses,
  getInputClasses,
  validateFieldValue,
  debounce,
  findFormContainer,
  findSiblingField,
  createFormProxy,
} from './core/field-utils';
export type { FieldState, FormValidationResult } from './core/field-utils';
export type { FormProxy } from './core/bc-setup';
