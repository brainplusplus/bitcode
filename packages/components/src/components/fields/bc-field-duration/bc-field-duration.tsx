import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getFieldClasses, validateFieldValue } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';

@Component({ tag: 'bc-field-duration', styleUrl: 'bc-field-duration.css', shadow: false })
export class BcFieldDuration {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: number = 0;
  @Prop() disabled: boolean = false;
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() placeholder: string = '';

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() clearable: boolean = false;
  @Prop() tooltip: string = '';
  @Prop() defaultValue: number = 0;
  @Prop() validateOn: ValidateOn | '' = '';

  @State() hours: number = 0;
  @State() minutes: number = 0;
  @State() private _fieldState: FieldState = createFieldState(0);
  customValidator?: (value: unknown) => string | null | Promise<string | null>;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;
  @Event() lcFieldFocus!: EventEmitter<FieldFocusEvent>;
  @Event() lcFieldBlur!: EventEmitter<FieldBlurEvent>;
  @Event() lcFieldClear!: EventEmitter<FieldClearEvent>;
  @Event() lcFieldInvalid!: EventEmitter<FieldValidationEvent>;
  @Event() lcFieldValid!: EventEmitter<FieldValidEvent>;

  componentWillLoad() {
    this._fieldState = createFieldState(this.value || this.defaultValue);
    if (!this.value && this.defaultValue) this.value = this.defaultValue;
    this.hours = Math.floor(this.value / 3600);
    this.minutes = Math.floor((this.value % 3600) / 60);
  }
  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }

  private update() {
    const old = this.value;
    this.value = this.hours * 3600 + this.minutes * 60;
    this._fieldState = markDirty(this._fieldState, this.value);
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
    if (this._getValidateOn() === 'change') this._runValidation();
  }
  private handleFocus() { this.lcFieldFocus.emit({ name: this.name, value: this.value }); }
  private handleBlur() { this._fieldState = markTouched(this._fieldState); this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true }); if (this._getValidateOn() === 'blur') this._runValidation(); }
  private handleClear() { const old = this.value; this.value = 0; this.hours = 0; this.minutes = 0; this._fieldState = markDirty(this._fieldState, 0); this.lcFieldClear.emit({ name: this.name, oldValue: old }); this.lcFieldChange.emit({ name: this.name, value: 0, oldValue: old }); }

  private async _runValidation(): Promise<ValidationResult> {
    this.validationStatus = 'validating';
    const result = await validateFieldValue(this.value, { required: this.required }, { customValidator: this.customValidator });
    if (result.valid) { this.validationStatus = 'valid'; this.validationMessage = ''; this.lcFieldValid.emit({ name: this.name, value: this.value }); }
    else { this.validationStatus = 'invalid'; this.validationMessage = result.errors[0] || ''; this.lcFieldInvalid.emit({ name: this.name, value: this.value, errors: result.errors }); }
    return result;
  }

  @Method() async validate(): Promise<ValidationResult> { return this._runValidation(); }
  @Method() async reset(): Promise<void> { this.value = this._fieldState.initialValue as number || this.defaultValue || 0; this.hours = Math.floor(this.value / 3600); this.minutes = Math.floor((this.value % 3600) / 60); this._fieldState = createFieldState(this.value); this.validationStatus = 'none'; this.validationMessage = ''; }
  @Method() async clear(): Promise<void> { this.handleClear(); }
  @Method() async setValue(value: number, emit: boolean = true): Promise<void> { const old = this.value; this.value = value; this.hours = Math.floor(value / 3600); this.minutes = Math.floor((value % 3600) / 60); this._fieldState = markDirty(this._fieldState, value); if (emit) this.lcFieldChange.emit({ name: this.name, value, oldValue: old }); }
  @Method() async getValue(): Promise<number> { return this.value; }
  @Method() async focusField(): Promise<void> { this.el.querySelector('input')?.focus(); }
  @Method() async blurField(): Promise<void> { this.el.querySelector('input')?.blur(); }
  @Method() async isDirty(): Promise<boolean> { return this._fieldState.dirty; }
  @Method() async isTouched(): Promise<boolean> { return this._fieldState.touched; }
  @Method() async setError(message: string): Promise<void> { this.validationStatus = 'invalid'; this.validationMessage = message; }
  @Method() async clearError(): Promise<void> { this.validationStatus = 'none'; this.validationMessage = ''; }

  render() {
    const fieldClasses = getFieldClasses({ size: this.size, validationStatus: this.validationStatus, disabled: this.disabled, readonly: this.readonly, dirty: this._fieldState.dirty, touched: this._fieldState.touched });
    const showError = this.validationStatus === 'invalid' && this.validationMessage;
    const showHint = this.hint && !showError;

    return (
      <div class={fieldClasses}>
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}{this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}</label>}
        <div class="bc-duration-inputs">
          <input type="number" class="bc-field-input bc-duration-part" value={this.hours} min={0} disabled={this.disabled || this.readonly} onInput={(e: Event) => { this.hours = Number((e.target as HTMLInputElement).value); this.update(); }} onFocus={() => this.handleFocus()} onBlur={() => this.handleBlur()} />
          <span class="bc-duration-sep">h</span>
          <input type="number" class="bc-field-input bc-duration-part" value={this.minutes} min={0} max={59} disabled={this.disabled || this.readonly} onInput={(e: Event) => { this.minutes = Number((e.target as HTMLInputElement).value); this.update(); }} onFocus={() => this.handleFocus()} onBlur={() => this.handleBlur()} />
          <span class="bc-duration-sep">m</span>
          {this.clearable && this.value > 0 && !this.disabled && !this.readonly && <button type="button" class="bc-field-clear-btn" onClick={() => this.handleClear()} tabIndex={-1}>&times;</button>}
        </div>
        {(showError || showHint) && (
          <div class="bc-field-footer">
            {showError && <div class="bc-field-error" role="alert">{this.validationMessage}</div>}
            {showHint && <div class="bc-field-hint">{this.hint}</div>}
          </div>
        )}
      </div>
    );
  }
}
