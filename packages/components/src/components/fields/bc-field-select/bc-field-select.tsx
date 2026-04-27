import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getAriaAttrs, getFieldClasses, getInputClasses, validateFieldValue } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';

@Component({ tag: 'bc-field-select', styleUrl: 'bc-field-select.css', shadow: false })
export class BcFieldSelect {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = 'Select...';
  @Prop() options: string = '[]';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() clearable: boolean = false;
  @Prop() tooltip: string = '';
  @Prop() loading: boolean = false;
  @Prop() defaultValue: string = '';
  @Prop() validateOn: ValidateOn | '' = '';
  @Prop() dependOn: string = '';
  @Prop() dataSource: string = '';

  @State() private _fieldState: FieldState = createFieldState('');
  private _selectEl?: HTMLSelectElement;
  private _dependListener?: (e: Event) => void;
  customValidator?: (value: unknown) => string | null | Promise<string | null>;
  validators?: Array<{ rule: string | ((value: unknown) => boolean | Promise<boolean>); message: string }>;
  serverValidator?: string | ((value: unknown) => Promise<string | null>);
  optionsFetcher?: (query: string, params: any) => Promise<unknown[]>;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;
  @Event() lcFieldFocus!: EventEmitter<FieldFocusEvent>;
  @Event() lcFieldBlur!: EventEmitter<FieldBlurEvent>;
  @Event() lcFieldClear!: EventEmitter<FieldClearEvent>;
  @Event() lcFieldInvalid!: EventEmitter<FieldValidationEvent>;
  @Event() lcFieldValid!: EventEmitter<FieldValidEvent>;

  componentWillLoad() { this._fieldState = createFieldState(this.value || this.defaultValue); if (!this.value && this.defaultValue) this.value = this.defaultValue; }
  componentDidLoad() { this._setupDependencyListener(); }
  disconnectedCallback() { this._cleanupDependencyListener(); }
  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }

  private _setupDependencyListener() {
    if (!this.dependOn) return;
    this._dependListener = (e: Event) => { const d = (e as CustomEvent<FieldChangeEvent>).detail; if (d && this.dependOn.split(',').map(s => s.trim()).includes(d.name)) { this.value = ''; this._fieldState = createFieldState(''); this.lcFieldChange.emit({ name: this.name, value: '', oldValue: d.value }); } };
    document.addEventListener('lcFieldChange', this._dependListener);
  }
  private _cleanupDependencyListener() { if (this._dependListener) { document.removeEventListener('lcFieldChange', this._dependListener); this._dependListener = undefined; } }

  private getOptions(): Array<string | { label: string; value: string }> { try { return JSON.parse(this.options); } catch { return []; } }
  private getOptLabel(opt: string | { label: string; value: string }): string { return typeof opt === 'string' ? opt : opt.label; }
  private getOptValue(opt: string | { label: string; value: string }): string { return typeof opt === 'string' ? opt : opt.value; }

  private handleChange(e: Event) {
    const old = this.value;
    this.value = (e.target as HTMLSelectElement).value;
    this._fieldState = markDirty(this._fieldState, this.value);
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
    if (this._getValidateOn() === 'change') this._runValidation();
  }
  private handleFocus() { this.lcFieldFocus.emit({ name: this.name, value: this.value }); }
  private handleBlur() { this._fieldState = markTouched(this._fieldState); this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true }); if (this._getValidateOn() === 'blur') this._runValidation(); }
  private handleClear() { const old = this.value; this.value = ''; this._fieldState = markDirty(this._fieldState, ''); this.lcFieldClear.emit({ name: this.name, oldValue: old }); this.lcFieldChange.emit({ name: this.name, value: '', oldValue: old }); }

  private async _runValidation(): Promise<ValidationResult> {
    this.validationStatus = 'validating';
    const result = await validateFieldValue(this.value, { required: this.required }, { validators: this.validators, customValidator: this.customValidator, serverValidator: this.serverValidator });
    if (result.valid) { this.validationStatus = 'valid'; this.validationMessage = ''; this.lcFieldValid.emit({ name: this.name, value: this.value }); }
    else { this.validationStatus = 'invalid'; this.validationMessage = result.errors[0] || ''; this.lcFieldInvalid.emit({ name: this.name, value: this.value, errors: result.errors }); }
    return result;
  }

  @Method() async validate(): Promise<ValidationResult> { return this._runValidation(); }
  @Method() async reset(): Promise<void> { this.value = this._fieldState.initialValue as string || this.defaultValue || ''; this._fieldState = createFieldState(this.value); this.validationStatus = 'none'; this.validationMessage = ''; }
  @Method() async clear(): Promise<void> { this.handleClear(); }
  @Method() async setValue(value: string, emit: boolean = true): Promise<void> { const old = this.value; this.value = value; this._fieldState = markDirty(this._fieldState, value); if (emit) this.lcFieldChange.emit({ name: this.name, value, oldValue: old }); }
  @Method() async getValue(): Promise<string> { return this.value; }
  @Method() async focusField(): Promise<void> { this._selectEl?.focus(); }
  @Method() async blurField(): Promise<void> { this._selectEl?.blur(); }
  @Method() async isDirty(): Promise<boolean> { return this._fieldState.dirty; }
  @Method() async isTouched(): Promise<boolean> { return this._fieldState.touched; }
  @Method() async setError(message: string): Promise<void> { this.validationStatus = 'invalid'; this.validationMessage = message; }
  @Method() async clearError(): Promise<void> { this.validationStatus = 'none'; this.validationMessage = ''; }
  @Method() async setOptions(options: unknown[]): Promise<void> { this.options = JSON.stringify(options); }

  render() {
    const fieldClasses = getFieldClasses({ size: this.size, validationStatus: this.validationStatus, disabled: this.disabled, readonly: this.readonly, loading: this.loading, dirty: this._fieldState.dirty, touched: this._fieldState.touched });
    const inputClasses = getInputClasses({ size: this.size, validationStatus: this.validationStatus });
    const ariaAttrs = getAriaAttrs({ name: this.name, required: this.required, disabled: this.disabled, readonly: this.readonly, validationStatus: this.validationStatus, validationMessage: this.validationMessage, hint: this.hint });
    const opts = this.getOptions();
    const showError = this.validationStatus === 'invalid' && this.validationMessage;
    const showHint = this.hint && !showError;

    return (
      <div class={fieldClasses}>
        {this.label && <label class="bc-field-label" htmlFor={this.name}>{this.label}{this.required && <span class="required">*</span>}{this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}</label>}
        <div class="bc-field-input-wrapper">
          <select ref={(el) => this._selectEl = el} id={this.name} class={inputClasses} name={this.name} disabled={this.disabled || this.readonly} required={this.required} onChange={(e) => this.handleChange(e)} onFocus={() => this.handleFocus()} onBlur={() => this.handleBlur()} {...ariaAttrs}>
            <option value="" disabled selected={!this.value}>{this.placeholder}</option>
            {opts.map(opt => <option value={this.getOptValue(opt)} selected={this.getOptValue(opt) === this.value}>{this.getOptLabel(opt)}</option>)}
          </select>
          {this.loading && <span class="bc-field-loading-indicator" />}
          {this.clearable && this.value && !this.disabled && !this.readonly && <button type="button" class="bc-field-clear-btn" onClick={() => this.handleClear()} tabIndex={-1}>&times;</button>}
        </div>
        <div class="bc-field-footer">
          {showError && <div class="bc-field-error" id={`${this.name}-error`} role="alert">{this.validationMessage}</div>}
          {showHint && <div class="bc-field-hint" id={`${this.name}-hint`}>{this.hint}</div>}
        </div>
      </div>
    );
  }
}
