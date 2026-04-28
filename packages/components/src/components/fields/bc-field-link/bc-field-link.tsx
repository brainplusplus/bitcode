import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getFieldClasses, validateFieldValue } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';
import { fetchOptions } from '../../../core/data-fetcher';

@Component({ tag: 'bc-field-link', styleUrl: 'bc-field-link.css', shadow: false })
export class BcFieldLink {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = 'Search...';
  @Prop() model: string = '';
  @Prop() displayField: string = 'name';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() options: string = '[]';
  @Prop() lookupColumns: string = '[]';

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() tooltip: string = '';
  @Prop() loading: boolean = false;
  @Prop() defaultValue: string = '';
  @Prop() validateOn: ValidateOn | '' = '';
  @Prop() dependOn: string = '';
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';

  @State() query: string = '';
  @State() results: Array<Record<string, unknown>> = [];
  @State() showDropdown: boolean = false;
  @State() displayValue: string = '';
  @State() showLookup: boolean = false;
  @State() private _fieldState: FieldState = createFieldState('');

  private debounceTimer: ReturnType<typeof setTimeout> | null = null;
  private _dependListener?: (e: Event) => void;
  customValidator?: (value: unknown) => string | null | Promise<string | null>;
  serverValidator?: string | ((value: unknown) => Promise<string | null>);

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;
  @Event() lcFieldFocus!: EventEmitter<FieldFocusEvent>;
  @Event() lcFieldBlur!: EventEmitter<FieldBlurEvent>;
  @Event() lcFieldClear!: EventEmitter<FieldClearEvent>;
  @Event() lcFieldInvalid!: EventEmitter<FieldValidationEvent>;
  @Event() lcFieldValid!: EventEmitter<FieldValidEvent>;

  componentWillLoad() { this._fieldState = createFieldState(this.value || this.defaultValue); this.displayValue = this.value; }
  componentDidLoad() { this._setupDependencyListener(); }
  disconnectedCallback() { this._cleanupDependencyListener(); }
  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }

  private _setupDependencyListener() {
    if (!this.dependOn) return;
    this._dependListener = (e: Event) => { const d = (e as CustomEvent<FieldChangeEvent>).detail; if (d && this.dependOn.split(',').map(s => s.trim()).includes(d.name)) { this.value = ''; this.displayValue = ''; this._fieldState = createFieldState(''); this.lcFieldChange.emit({ name: this.name, value: '', oldValue: d.value }); } };
    document.addEventListener('lcFieldChange', this._dependListener);
  }
  private _cleanupDependencyListener() { if (this._dependListener) { document.removeEventListener('lcFieldChange', this._dependListener); this._dependListener = undefined; } }

  private async search(q: string) {
    this.query = q;
    if (this.debounceTimer) clearTimeout(this.debounceTimer);
    if (q.length < 1) { this.results = []; this.showDropdown = false; return; }
    this.debounceTimer = setTimeout(async () => {
      try {
        const items = await fetchOptions({ element: this.el, dataSource: this.dataSource, model: this.model, query: q, fetchHeaders: this.fetchHeaders || undefined }) as Array<Record<string, unknown>>;
        this.results = items;
        this.showDropdown = items.length > 0;
      } catch { this.results = []; this.showDropdown = false; }
    }, 300);
  }

  private select(item: Record<string, unknown>) {
    const old = this.value;
    this.value = String(item['id'] || '');
    this.displayValue = String(item[this.displayField] || item['name'] || item['id'] || '');
    this.query = '';
    this.showDropdown = false;
    this.results = [];
    this._fieldState = markDirty(this._fieldState, this.value);
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  private handleClear() {
    const old = this.value;
    this.value = '';
    this.displayValue = '';
    this._fieldState = markDirty(this._fieldState, '');
    this.lcFieldClear.emit({ name: this.name, oldValue: old });
    this.lcFieldChange.emit({ name: this.name, value: '', oldValue: old });
  }

  private handleLookupSelect(e: CustomEvent) {
    const records = e.detail.records;
    if (records && records.length > 0) this.select(records[0]);
    this.showLookup = false;
  }

  private handleFocus() { this.lcFieldFocus.emit({ name: this.name, value: this.value }); }
  private handleBlur() { this._fieldState = markTouched(this._fieldState); this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true }); if (this._getValidateOn() === 'blur') this._runValidation(); }

  private async _runValidation(): Promise<ValidationResult> {
    this.validationStatus = 'validating';
    const result = await validateFieldValue(this.value, { required: this.required }, { customValidator: this.customValidator, serverValidator: this.serverValidator });
    if (result.valid) { this.validationStatus = 'valid'; this.validationMessage = ''; this.lcFieldValid.emit({ name: this.name, value: this.value }); }
    else { this.validationStatus = 'invalid'; this.validationMessage = result.errors[0] || ''; this.lcFieldInvalid.emit({ name: this.name, value: this.value, errors: result.errors }); }
    return result;
  }

  @Method() async validate(): Promise<ValidationResult> { return this._runValidation(); }
  @Method() async reset(): Promise<void> { this.value = this._fieldState.initialValue as string || this.defaultValue || ''; this.displayValue = this.value; this._fieldState = createFieldState(this.value); this.validationStatus = 'none'; this.validationMessage = ''; }
  @Method() async clear(): Promise<void> { this.handleClear(); }
  @Method() async setValue(value: string, emit: boolean = true): Promise<void> { const old = this.value; this.value = value; this._fieldState = markDirty(this._fieldState, value); if (emit) this.lcFieldChange.emit({ name: this.name, value, oldValue: old }); }
  @Method() async getValue(): Promise<string> { return this.value; }
  @Method() async focusField(): Promise<void> { this.el.querySelector('input')?.focus(); }
  @Method() async blurField(): Promise<void> { this.el.querySelector('input')?.blur(); }
  @Method() async isDirty(): Promise<boolean> { return this._fieldState.dirty; }
  @Method() async isTouched(): Promise<boolean> { return this._fieldState.touched; }
  @Method() async setError(message: string): Promise<void> { this.validationStatus = 'invalid'; this.validationMessage = message; }
  @Method() async clearError(): Promise<void> { this.validationStatus = 'none'; this.validationMessage = ''; }

  render() {
    const fieldClasses = getFieldClasses({ size: this.size, validationStatus: this.validationStatus, disabled: this.disabled, readonly: this.readonly, loading: this.loading, dirty: this._fieldState.dirty, touched: this._fieldState.touched });
    const showError = this.validationStatus === 'invalid' && this.validationMessage;
    const showHint = this.hint && !showError;

    return (
      <div class={{ ...fieldClasses, 'bc-link-wrap': true }}>
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}{this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}</label>}
        <div class="bc-link-input-wrap">
          {this.value ? (
            <div class="bc-link-selected">
              <span class="bc-link-display">{this.displayValue}</span>
              {!this.readonly && !this.disabled && <button type="button" class="bc-link-clear" onClick={() => this.handleClear()}>{'\u00D7'}</button>}
            </div>
          ) : (
            <div class="bc-link-input-row">
              <input type="text" class="bc-field-input" placeholder={this.placeholder} readOnly={this.readonly} disabled={this.disabled} value={this.query} onInput={(e: Event) => this.search((e.target as HTMLInputElement).value)} onFocus={() => { this.handleFocus(); if (this.results.length > 0) this.showDropdown = true; }} onBlur={() => { setTimeout(() => { this.showDropdown = false; }, 200); this.handleBlur(); }} />
              {!this.readonly && !this.disabled && <button type="button" class="bc-link-lookup-btn" onClick={() => { this.showLookup = true; }} title={'Browse ' + this.model}>{'\uD83D\uDD0D'}</button>}
            </div>
          )}
          {this.loading && <span class="bc-field-loading-indicator" />}
          {this.showDropdown && (
            <div class="bc-link-dropdown">
              {this.results.map(item => (
                <div class="bc-link-option" onMouseDown={() => this.select(item)}>{String(item[this.displayField] || item['name'] || item['id'] || '')}</div>
              ))}
            </div>
          )}
        </div>
        <div class="bc-field-footer">
          {showError && <div class="bc-field-error" role="alert">{this.validationMessage}</div>}
          {showHint && <div class="bc-field-hint">{this.hint}</div>}
        </div>
        <bc-lookup-modal open={this.showLookup} model={this.model} display-field={this.displayField} columns={this.lookupColumns} onLcLookupSelect={(e: CustomEvent) => this.handleLookupSelect(e)} onLcLookupClose={() => { this.showLookup = false; }}></bc-lookup-modal>
      </div>
    );
  }
}
