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
