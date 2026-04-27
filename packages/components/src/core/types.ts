// ============================================================================
// EXISTING EVENT INTERFACES
// ============================================================================

export interface FieldChangeEvent {
  name: string;
  value: unknown;
  oldValue: unknown;
}

export interface FormSubmitEvent {
  model: string;
  data: Record<string, unknown>;
  id?: string;
}

export interface ActionClickEvent {
  process: string;
  recordId?: string;
}

export interface ListParams {
  page?: number;
  pageSize?: number;
  sort?: string;
  order?: 'asc' | 'desc';
  filters?: Record<string, unknown>;
  q?: string;
}

export interface ListResponse {
  data: Record<string, unknown>[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface KanbanMoveEvent {
  id: string;
  from: string;
  to: string;
}

export interface RowSelectEvent {
  ids: string[];
}

export interface DialogEvent {
  type: string;
  result?: unknown;
}

export interface ToastEvent {
  type: 'success' | 'error' | 'warning' | 'info';
  message: string;
  duration?: number;
}

// ============================================================================
// FIELD EVENT INTERFACES (Phase 2: universal field events)
// ============================================================================

export interface FieldFocusEvent {
  name: string;
  value: unknown;
}

export interface FieldBlurEvent {
  name: string;
  value: unknown;
  dirty: boolean;
  touched: boolean;
}

export interface FieldClearEvent {
  name: string;
  oldValue: unknown;
}

export interface FieldValidationEvent {
  name: string;
  value: unknown;
  errors: string[];
}

export interface FieldValidEvent {
  name: string;
  value: unknown;
}

// ============================================================================
// DATA FETCH EVENT INTERFACES (Phase 3-5: 4-level data strategy)
// ============================================================================

export interface BeforeFetchEvent {
  url: string;
  headers: Record<string, string>;
  params: FetchParams;
}

export interface AfterFetchEvent {
  response: unknown;
  data: unknown[] | null;
  total: number;
}

// ============================================================================
// SELECT/DROPDOWN EVENT INTERFACES (Phase 3: select-family)
// ============================================================================

export interface OptionsLoadEvent {
  options: unknown[];
  total: number;
}

export interface OptionsErrorEvent {
  error: string;
}

export interface OptionCreateEvent {
  value: string;
}

// ============================================================================
// DATATABLE EVENT INTERFACES (Phase 4: datatable)
// ============================================================================

export interface RowEditEvent {
  row: Record<string, unknown>;
  field: string;
  value: unknown;
  oldValue: unknown;
}

export interface RowExpandEvent {
  row: Record<string, unknown>;
  expanded: boolean;
}

export interface CellClickEvent {
  row: Record<string, unknown>;
  column: string;
  value: unknown;
}

export interface ColumnResizeEvent {
  column: string;
  width: number;
}

export interface ColumnReorderEvent {
  columns: string[];
}

export interface SortChangeEvent {
  sorts: Array<{ field: string; direction: 'asc' | 'desc' }>;
}

export interface FilterChangeEvent {
  filters: Record<string, unknown>;
}

export interface PageChangeEvent {
  page: number;
  pageSize: number;
}

export interface ScrollEndEvent {
  direction: 'bottom' | 'top';
}

// ============================================================================
// CHART EVENT INTERFACES (Phase 5: charts)
// ============================================================================

export interface ChartClickEvent {
  name: string;
  value: unknown;
  dataIndex: number;
}

export interface ChartHoverEvent {
  name: string;
  value: unknown;
  dataIndex: number;
}

// ============================================================================
// DATA FETCHING TYPES (used by data-fetcher.ts and components)
// ============================================================================

export interface FetchParams {
  page?: number;
  pageSize?: number;
  sort?: Array<{ field: string; direction: 'asc' | 'desc' }>;
  filters?: Record<string, unknown>;
  search?: string;
  dependValues?: Record<string, unknown>;
}

export interface FetchResult {
  data: unknown[];
  total: number;
}

export type DataFetcher = (params: FetchParams) => Promise<FetchResult>;
export type OptionsFetcher = (query: string, params: FetchParams) => Promise<unknown[]>;

// ============================================================================
// VALIDATION TYPES (used by validation-engine.ts and field-utils.ts)
// ============================================================================

export interface ValidationResult {
  valid: boolean;
  errors: string[];
}

export interface ValidationRule {
  rule: string | ((value: unknown) => boolean | Promise<boolean>);
  message: string;
}

export type ValidateOn = 'blur' | 'change' | 'submit' | 'manual';

// ============================================================================
// FIELD TYPES & CONFIG (existing, unchanged)
// ============================================================================

export type FieldType =
  | 'string'
  | 'smalltext'
  | 'text'
  | 'richtext'
  | 'markdown'
  | 'html'
  | 'code'
  | 'password'
  | 'integer'
  | 'float'
  | 'decimal'
  | 'currency'
  | 'percent'
  | 'boolean'
  | 'toggle'
  | 'selection'
  | 'radio'
  | 'many2one'
  | 'one2many'
  | 'many2many'
  | 'dynamic_link'
  | 'many2many_check'
  | 'table_multiselect'
  | 'date'
  | 'time'
  | 'datetime'
  | 'duration'
  | 'file'
  | 'image'
  | 'signature'
  | 'barcode'
  | 'color'
  | 'geolocation'
  | 'rating'
  | 'json'
  | 'computed';

export type WidgetType =
  | 'statusbar'
  | 'priority'
  | 'handle'
  | 'badge'
  | 'copy'
  | 'phone'
  | 'email'
  | 'url'
  | 'progress'
  | 'domain';

export interface FieldBehavior {
  dependsOn?: string;
  readonlyIf?: string;
  mandatoryIf?: string;
  fetchFrom?: string;
  formula?: string;
}

export interface FieldConfig {
  type: FieldType;
  label?: string;
  required?: boolean;
  readonly?: boolean;
  widget?: WidgetType;
  behavior?: FieldBehavior;
  options?: string[];
  model?: string;
  default?: unknown;
  placeholder?: string;
}

// ============================================================================
// BCSETUP CONFIG TYPES (used by bc-setup.ts)
// ============================================================================

export interface BcAuthConfig {
  type: 'bearer' | 'header' | 'cookie' | 'none';
  token?: string | (() => string | null);
  headerName?: string;
  headerValue?: string | (() => string | null);
}

export interface BcConfig {
  baseUrl: string;
  headers: Record<string, string | (() => string)>;
  auth: BcAuthConfig;
  responseTransformer?: (response: unknown) => FetchResult;
  validateOn: ValidateOn;
  validationMessages: Record<string, string>;
  size: 'sm' | 'md' | 'lg';
  locale: string;
  theme: 'light' | 'dark' | 'system' | string;
}
