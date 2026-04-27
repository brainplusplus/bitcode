import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getAriaAttrs, getFieldClasses, getInputClasses, validateFieldValue, debounce } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';

@Component({ tag: 'bc-field-percent', styleUrl: 'bc-field-percent.css', shadow: false })
export class BcFieldPercent {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: number = 0;
  @Prop() placeholder: string = '';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() min: number = 0;
  @Prop() max: number = 100;
  @Prop() step: number = 1;

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() clearable: boolean = false;
  @Prop() prefixText: string = '';
  @Prop() suffixText: string = '';
  @Prop() tooltip: string = '';
  @Prop() loading: boolean = false;
  @Prop() autofocus: boolean = false;
  @Prop() defaultValue: number = 0;
  @Prop() validateOn: ValidateOn | '' = '';

  @State() private _fieldState: FieldState = createFieldState(0);
  private _inputEl?: HTMLInputElement;
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
  componentDidLoad() { if (this.autofocus && this._inputEl) this._inputEl.focus(); }
  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }

  private handleInput(e: Event) { const old = this.value; this.value = Number((e.target as HTMLInputElement).value); this._fieldState = markDirty(this._fieldState, this.value); this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old }); if (this._getValidateOn() === 'change') debounce(`validate-${this.name}`, () => this._runValidation(), 300); }
  private handleFocus() { this.lcFieldFocus.emit({ name: this.name, value: this.value }); }
  private handleBlur() { this._fieldState = markTouched(this._fieldState); this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true }); if (this._getValidateOn() === 'blur') this._runValidation(); }
  private handleClear() { const old = this.value; this.value = 0; this._fieldState = markDirty(this._fieldState, 0); this.lcFieldClear.emit({ name: this.name, oldValue: old }); this.lcFieldChange.emit({ name: this.name, value: 0, oldValue: old }); this._inputEl?.focus(); }

  private async _runValidation(): Promise<ValidationResult> {
    this.validationStatus = 'validating';
    const result = await validateFieldValue(this.value, { required: this.required, min: this.min, max: this.max }, { validators: this.validators, customValidator: this.customValidator, serverValidator: this.serverValidator });
    if (result.valid) { this.validationStatus = 'valid'; this.validationMessage = ''; this.lcFieldValid.emit({ name: this.name, value: this.value }); }
    else { this.validationStatus = 'invalid'; this.validationMessage = result.errors[0] || ''; this.lcFieldInvalid.emit({ name: this.name, value: this.value, errors: result.errors }); }
    return result;
  }

  @Method() async validate(): Promise<ValidationResult> { return this._runValidation(); }
  @Method() async reset(): Promise<void> { this.value = this._fieldState.initialValue as number || this.defaultValue || 0; this._fieldState = createFieldState(this.value); this.validationStatus = 'none'; this.validationMessage = ''; }
  @Method() async clear(): Promise<void> { this.handleClear(); }
  @Method() async setValue(value: number, emit: boolean = true): Promise<void> { const old = this.value; this.value = value; this._fieldState = markDirty(this._fieldState, value); if (emit) this.lcFieldChange.emit({ name: this.name, value, oldValue: old }); }
  @Method() async getValue(): Promise<number> { return this.value; }
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
          {this.prefixText && <span class="bc-field-prefix">{this.prefixText}</span>}
          <input ref={(el) => this._inputEl = el} id={this.name} type="number" class={inputClasses} name={this.name} value={this.value} placeholder={this.placeholder} required={this.required} readOnly={this.readonly} disabled={this.disabled} min={this.min} max={this.max} step={this.step} onInput={(e: Event) => this.handleInput(e)} onFocus={() => this.handleFocus()} onBlur={() => this.handleBlur()} {...ariaAttrs} />
          {this.loading && <span class="bc-field-loading-indicator" />}
          {this.clearable && this.value !== 0 && !this.disabled && !this.readonly && <button type="button" class="bc-field-clear-btn" onClick={() => this.handleClear()} tabIndex={-1}>&times;</button>}
          <span class="bc-field-suffix">{this.suffixText || '%'}</span>
        </div>
        <div class="bc-field-footer">
          {showError && <div class="bc-field-error" id={`${this.name}-error`} role="alert">{this.validationMessage}</div>}
          {showHint && <div class="bc-field-hint" id={`${this.name}-hint`}>{this.hint}</div>}
        </div>
      </div>
    );
  }
}
