import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getFieldClasses, validateFieldValue } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';
import { fetchOptions } from '../../../core/data-fetcher';

@Component({ tag: 'bc-field-tableselect', styleUrl: 'bc-field-tableselect.css', shadow: false })
export class BcFieldTableselect {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '[]';
  @Prop() placeholder: string = 'Search and add...';
  @Prop() model: string = '';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() options: string = '[]';

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() clearable: boolean = false;
  @Prop() tooltip: string = '';
  @Prop() loading: boolean = false;
  @Prop() defaultValue: string = '[]';
  @Prop() validateOn: ValidateOn | '' = '';

  @State() query: string = '';
  @State() results: Array<Record<string, unknown>> = [];
  @State() showDropdown: boolean = false;
  @State() private _fieldState: FieldState = createFieldState('[]');

  private debounceTimer: ReturnType<typeof setTimeout> | null = null;
  customValidator?: (value: unknown) => string | null | Promise<string | null>;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;
  @Event() lcFieldFocus!: EventEmitter<FieldFocusEvent>;
  @Event() lcFieldBlur!: EventEmitter<FieldBlurEvent>;
  @Event() lcFieldClear!: EventEmitter<FieldClearEvent>;
  @Event() lcFieldInvalid!: EventEmitter<FieldValidationEvent>;
  @Event() lcFieldValid!: EventEmitter<FieldValidEvent>;

  componentWillLoad() { this._fieldState = createFieldState(this.value || this.defaultValue); }
  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }
  private getValues(): string[] { try { return JSON.parse(this.value); } catch { return []; } }

  private async search(q: string) {
    this.query = q;
    if (this.debounceTimer) clearTimeout(this.debounceTimer);
    if (q.length < 1) { this.results = []; this.showDropdown = false; return; }
    this.debounceTimer = setTimeout(async () => {
      try { this.results = await fetchOptions({ element: this.el, model: this.model, query: q }) as Array<Record<string, unknown>>; this.showDropdown = this.results.length > 0; }
      catch { this.results = []; this.showDropdown = false; }
    }, 300);
  }

  private addItem(val: string) {
    const items = this.getValues();
    if (items.includes(val)) return;
    const old = this.value;
    items.push(val);
    this.value = JSON.stringify(items);
    this.query = '';
    this.showDropdown = false;
    this._fieldState = markDirty(this._fieldState, this.value);
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  private removeItem(val: string) {
    const old = this.value;
    this.value = JSON.stringify(this.getValues().filter(v => v !== val));
    this._fieldState = markDirty(this._fieldState, this.value);
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  private handleClear() { const old = this.value; this.value = '[]'; this._fieldState = markDirty(this._fieldState, '[]'); this.lcFieldClear.emit({ name: this.name, oldValue: old }); this.lcFieldChange.emit({ name: this.name, value: '[]', oldValue: old }); }
  private handleFocus() { this.lcFieldFocus.emit({ name: this.name, value: this.value }); }
  private handleBlur() { this._fieldState = markTouched(this._fieldState); this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true }); if (this._getValidateOn() === 'blur') this._runValidation(); }

  private async _runValidation(): Promise<ValidationResult> {
    this.validationStatus = 'validating';
    const vals = this.getValues();
    const result = await validateFieldValue(vals.length > 0 ? this.value : '', { required: this.required }, { customValidator: this.customValidator });
    if (result.valid) { this.validationStatus = 'valid'; this.validationMessage = ''; this.lcFieldValid.emit({ name: this.name, value: this.value }); }
    else { this.validationStatus = 'invalid'; this.validationMessage = result.errors[0] || ''; this.lcFieldInvalid.emit({ name: this.name, value: this.value, errors: result.errors }); }
    return result;
  }

  @Method() async validate(): Promise<ValidationResult> { return this._runValidation(); }
  @Method() async reset(): Promise<void> { this.value = this._fieldState.initialValue as string || this.defaultValue || '[]'; this._fieldState = createFieldState(this.value); this.validationStatus = 'none'; this.validationMessage = ''; }
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
    const items = this.getValues();
    const showError = this.validationStatus === 'invalid' && this.validationMessage;
    const showHint = this.hint && !showError;

    return (
      <div class={{ ...fieldClasses, 'bc-tableselect-wrap': true }}>
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}{this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}</label>}
        {items.length > 0 && (
          <div class="bc-tableselect-list">
            {items.map(item => (
              <div class="bc-tableselect-item"><span>{item}</span>{!this.readonly && !this.disabled && <button type="button" class="bc-tableselect-remove" onClick={() => this.removeItem(item)}>&times;</button>}</div>
            ))}
          </div>
        )}
        {!this.readonly && !this.disabled && (
          <div class="bc-tableselect-search">
            <input type="text" class="bc-field-input" placeholder={this.placeholder} value={this.query} onInput={(e: Event) => this.search((e.target as HTMLInputElement).value)} onFocus={() => this.handleFocus()} onBlur={() => { setTimeout(() => { this.showDropdown = false; }, 200); this.handleBlur(); }} />
            {this.showDropdown && (
              <div class="bc-tableselect-dropdown">
                {this.results.map(item => { const val = String(item['name'] || item['id'] || ''); return <div class="bc-tableselect-option" onMouseDown={() => this.addItem(val)}>{val}</div>; })}
              </div>
            )}
          </div>
        )}
        {this.clearable && items.length > 0 && !this.disabled && !this.readonly && <button type="button" class="bc-field-clear-btn" onClick={() => this.handleClear()} tabIndex={-1}>&times; Clear all</button>}
        <div class="bc-field-footer">
          {showError && <div class="bc-field-error" role="alert">{this.validationMessage}</div>}
          {showHint && <div class="bc-field-hint">{this.hint}</div>}
        </div>
      </div>
    );
  }
}
