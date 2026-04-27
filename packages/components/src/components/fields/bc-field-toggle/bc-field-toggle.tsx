import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getFieldClasses, validateFieldValue } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';

@Component({ tag: 'bc-field-toggle', styleUrl: 'bc-field-toggle.css', shadow: false })
export class BcFieldToggle {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() tooltip: string = '';
  @Prop() defaultValue: boolean = false;
  @Prop() validateOn: ValidateOn | '' = '';

  @State() private _fieldState: FieldState = createFieldState(false);
  customValidator?: (value: unknown) => string | null | Promise<string | null>;
  validators?: Array<{ rule: string | ((value: unknown) => boolean | Promise<boolean>); message: string }>;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;
  @Event() lcFieldFocus!: EventEmitter<FieldFocusEvent>;
  @Event() lcFieldBlur!: EventEmitter<FieldBlurEvent>;
  @Event() lcFieldClear!: EventEmitter<FieldClearEvent>;
  @Event() lcFieldInvalid!: EventEmitter<FieldValidationEvent>;
  @Event() lcFieldValid!: EventEmitter<FieldValidEvent>;

  componentWillLoad() { this._fieldState = createFieldState(this.value || this.defaultValue); }
  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }

  private handleClick() {
    if (this.disabled || this.readonly) return;
    const old = this.value;
    this.value = !this.value;
    this._fieldState = markDirty(this._fieldState, this.value);
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
    if (this._getValidateOn() === 'change') this._runValidation();
  }
  private handleFocus() { this.lcFieldFocus.emit({ name: this.name, value: this.value }); }
  private handleBlur() { this._fieldState = markTouched(this._fieldState); this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true }); if (this._getValidateOn() === 'blur') this._runValidation(); }

  private async _runValidation(): Promise<ValidationResult> {
    this.validationStatus = 'validating';
    const result = await validateFieldValue(this.value, { required: this.required }, { validators: this.validators, customValidator: this.customValidator });
    if (result.valid) { this.validationStatus = 'valid'; this.validationMessage = ''; this.lcFieldValid.emit({ name: this.name, value: this.value }); }
    else { this.validationStatus = 'invalid'; this.validationMessage = result.errors[0] || ''; this.lcFieldInvalid.emit({ name: this.name, value: this.value, errors: result.errors }); }
    return result;
  }

  @Method() async validate(): Promise<ValidationResult> { return this._runValidation(); }
  @Method() async reset(): Promise<void> { this.value = this._fieldState.initialValue as boolean || this.defaultValue || false; this._fieldState = createFieldState(this.value); this.validationStatus = 'none'; this.validationMessage = ''; }
  @Method() async clear(): Promise<void> { const old = this.value; this.value = false; this._fieldState = markDirty(this._fieldState, false); this.lcFieldClear.emit({ name: this.name, oldValue: old }); this.lcFieldChange.emit({ name: this.name, value: false, oldValue: old }); }
  @Method() async setValue(value: boolean, emit: boolean = true): Promise<void> { const old = this.value; this.value = value; this._fieldState = markDirty(this._fieldState, value); if (emit) this.lcFieldChange.emit({ name: this.name, value, oldValue: old }); }
  @Method() async getValue(): Promise<boolean> { return this.value; }
  @Method() async focusField(): Promise<void> { this.el.querySelector('button')?.focus(); }
  @Method() async blurField(): Promise<void> { this.el.querySelector('button')?.blur(); }
  @Method() async isDirty(): Promise<boolean> { return this._fieldState.dirty; }
  @Method() async isTouched(): Promise<boolean> { return this._fieldState.touched; }
  @Method() async setError(message: string): Promise<void> { this.validationStatus = 'invalid'; this.validationMessage = message; }
  @Method() async clearError(): Promise<void> { this.validationStatus = 'none'; this.validationMessage = ''; }

  render() {
    const fieldClasses = getFieldClasses({ size: this.size, validationStatus: this.validationStatus, disabled: this.disabled, readonly: this.readonly, dirty: this._fieldState.dirty, touched: this._fieldState.touched });
    const showError = this.validationStatus === 'invalid' && this.validationMessage;
    const showHint = this.hint && !showError;

    return (
      <div class={{ ...fieldClasses, 'bc-field-inline': true }}>
        {this.label && <span class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}{this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}</span>}
        <button type="button" class={{ 'bc-toggle': true, 'active': this.value }} disabled={this.disabled || this.readonly} onClick={() => this.handleClick()} onFocus={() => this.handleFocus()} onBlur={() => this.handleBlur()} aria-pressed={String(this.value)}>
          <span class="bc-toggle-knob"></span>
        </button>
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
