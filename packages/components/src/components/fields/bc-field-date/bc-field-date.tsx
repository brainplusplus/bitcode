import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getAriaAttrs, getFieldClasses, getInputClasses, validateFieldValue, debounce } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';

@Component({ tag: 'bc-field-date', styleUrl: 'bc-field-date.css', shadow: false })
export class BcFieldDate {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = '';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() min: string = '';
  @Prop() max: string = '';

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() clearable: boolean = false;
  @Prop() tooltip: string = '';
  @Prop() loading: boolean = false;
  @Prop() autofocus: boolean = false;
  @Prop() defaultValue: string = '';
  @Prop() validateOn: ValidateOn | '' = '';
  @Prop() dependOn: string = '';

  @State() private _fieldState: FieldState = createFieldState('');
  private _inputEl?: HTMLInputElement;
  private _dependListener?: (e: Event) => void;
  customValidator?: (value: unknown) => string | null | Promise<string | null>;
  validators?: Array<{ rule: string | ((value: unknown) => boolean | Promise<boolean>); message: string }>;
  serverValidator?: string | ((value: unknown) => Promise<string | null>);

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;
  @Event() lcFieldFocus!: EventEmitter<FieldFocusEvent>;
  @Event() lcFieldBlur!: EventEmitter<FieldBlurEvent>;
  @Event() lcFieldClear!: EventEmitter<FieldClearEvent>;
  @Event() lcFieldInvalid!: EventEmitter<FieldValidationEvent>;
  @Event() lcFieldValid!: EventEmitter<FieldValidEvent>;

  componentWillLoad() { this._fieldState = createFieldState(this.value || this.defaultValue); if (!this.value && this.defaultValue) this.value = this.defaultValue; }
  componentDidLoad() { if (this.autofocus && this._inputEl) this._inputEl.focus(); this._setupDependencyListener(); }
  disconnectedCallback() { this._cleanupDependencyListener(); }

  private _setupDependencyListener() {
    if (!this.dependOn) return;
    this._dependListener = (e: Event) => { const d = (e as CustomEvent<FieldChangeEvent>).detail; if (d && this.dependOn.split(',').map(s => s.trim()).includes(d.name)) { this.value = ''; this._fieldState = createFieldState(''); this.lcFieldChange.emit({ name: this.name, value: '', oldValue: d.value }); } };
    document.addEventListener('lcFieldChange', this._dependListener);
  }
  private _cleanupDependencyListener() { if (this._dependListener) { document.removeEventListener('lcFieldChange', this._dependListener); this._dependListener = undefined; } }
  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }

  private handleInput(e: Event) { const old = this.value; this.value = (e.target as HTMLInputElement).value; this._fieldState = markDirty(this._fieldState, this.value); this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old }); if (this._getValidateOn() === 'change') debounce(`validate-${this.name}`, () => this._runValidation(), 300); }
  private handleFocus() { this.lcFieldFocus.emit({ name: this.name, value: this.value }); }
  private handleBlur() { this._fieldState = markTouched(this._fieldState); this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true }); if (this._getValidateOn() === 'blur') this._runValidation(); }
  private handleClear() { const old = this.value; this.value = ''; this._fieldState = markDirty(this._fieldState, ''); this.lcFieldClear.emit({ name: this.name, oldValue: old }); this.lcFieldChange.emit({ name: this.name, value: '', oldValue: old }); this._inputEl?.focus(); }

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
  @Method() async focusField(): Promise<void> { this._inputEl?.focus(); }
  @Method() async blurField(): Promise<void> { this._inputEl?.blur(); }
  @Method() async isDirty(): Promise<boolean> { return this._fieldState.dirty; }
  @Method() async isTouched(): Promise<boolean> { return this._fieldState.touched; }
  @Method() async setError(message: string): Promise<void> { this.validationStatus = 'invalid'; this.validationMessage = message; }
  @Method() async clearError(): Promise<void> { this.validationStatus = 'none'; this.validationMessage = ''; }

  render() {
    const fieldClasses = getFieldClasses({ size: this.size, validationStatus: this.validationStatus, disabled: this.disabled, readonly: this.readonly, loading: this.loading, dirty: this._fieldState.dirty, touched: this._fieldState.touched });
    const inputClasses = getInputClasses({ size: this.size, validationStatus: this.validationStatus });
    const ariaAttrs = getAriaAttrs({ name: this.name, required: this.required, disabled: this.disabled, readonly: this.readonly, validationStatus: this.validationStatus, validationMessage: this.validationMessage, hint: this.hint });
    const showError = this.validationStatus === 'invalid' && this.validationMessage;
    const showHint = this.hint && !showError;

    return (
      <div class={fieldClasses}>
        {this.label && <label class="bc-field-label" htmlFor={this.name}>{this.label}{this.required && <span class="required">*</span>}{this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}</label>}
        <div class="bc-field-input-wrapper">
          <input ref={(el) => this._inputEl = el} id={this.name} type="date" class={inputClasses} name={this.name} value={this.value} placeholder={this.placeholder} required={this.required} readOnly={this.readonly} disabled={this.disabled} min={this.min || undefined} max={this.max || undefined} onInput={(e: Event) => this.handleInput(e)} onFocus={() => this.handleFocus()} onBlur={() => this.handleBlur()} {...ariaAttrs} />
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
