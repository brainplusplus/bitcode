import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getAriaAttrs, getFieldClasses, getInputClasses, validateFieldValue, debounce } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';

@Component({ tag: 'bc-field-smalltext', styleUrl: 'bc-field-smalltext.css', shadow: false })
export class BcFieldSmalltext {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = '';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() rows: number = 3;

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() minLength: number = 0;
  @Prop() maxLength: number = 0;
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() clearable: boolean = false;
  @Prop() tooltip: string = '';
  @Prop() showCount: boolean = false;
  @Prop() loading: boolean = false;
  @Prop() autofocus: boolean = false;
  @Prop() defaultValue: string = '';
  @Prop() validateOn: ValidateOn | '' = '';

  @State() private _fieldState: FieldState = createFieldState('');
  private _textareaEl?: HTMLTextAreaElement;
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
  componentDidLoad() { if (this.autofocus && this._textareaEl) this._textareaEl.focus(); }
  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }

  private handleInput(e: Event) { const old = this.value; this.value = (e.target as HTMLTextAreaElement).value; this._fieldState = markDirty(this._fieldState, this.value); this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old }); if (this._getValidateOn() === 'change') debounce(`validate-${this.name}`, () => this._runValidation(), 300); }
  private handleFocus() { this.lcFieldFocus.emit({ name: this.name, value: this.value }); }
  private handleBlur() { this._fieldState = markTouched(this._fieldState); this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true }); if (this._getValidateOn() === 'blur') this._runValidation(); }
  private handleClear() { const old = this.value; this.value = ''; this._fieldState = markDirty(this._fieldState, ''); this.lcFieldClear.emit({ name: this.name, oldValue: old }); this.lcFieldChange.emit({ name: this.name, value: '', oldValue: old }); this._textareaEl?.focus(); }

  private async _runValidation(): Promise<ValidationResult> {
    this.validationStatus = 'validating';
    const result = await validateFieldValue(this.value, { required: this.required, minLength: this.minLength, maxLength: this.maxLength }, { validators: this.validators, customValidator: this.customValidator, serverValidator: this.serverValidator });
    if (result.valid) { this.validationStatus = 'valid'; this.validationMessage = ''; this.lcFieldValid.emit({ name: this.name, value: this.value }); }
    else { this.validationStatus = 'invalid'; this.validationMessage = result.errors[0] || ''; this.lcFieldInvalid.emit({ name: this.name, value: this.value, errors: result.errors }); }
    return result;
  }

  @Method() async validate(): Promise<ValidationResult> { return this._runValidation(); }
  @Method() async reset(): Promise<void> { this.value = this._fieldState.initialValue as string || this.defaultValue || ''; this._fieldState = createFieldState(this.value); this.validationStatus = 'none'; this.validationMessage = ''; }
  @Method() async clear(): Promise<void> { this.handleClear(); }
  @Method() async setValue(value: string, emit: boolean = true): Promise<void> { const old = this.value; this.value = value; this._fieldState = markDirty(this._fieldState, value); if (emit) this.lcFieldChange.emit({ name: this.name, value, oldValue: old }); }
  @Method() async getValue(): Promise<string> { return this.value; }
  @Method() async focusField(): Promise<void> { this._textareaEl?.focus(); }
  @Method() async blurField(): Promise<void> { this._textareaEl?.blur(); }
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
    const currentLength = (this.value || '').length;
    const maxLen = this.maxLength || 0;

    return (
      <div class={fieldClasses}>
        {this.label && <label class="bc-field-label" htmlFor={this.name}>{this.label}{this.required && <span class="required">*</span>}{this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}</label>}
        <div class="bc-field-input-wrapper" style={{ position: 'relative' }}>
          <textarea ref={(el) => this._textareaEl = el} id={this.name} class={{ ...inputClasses, 'bc-field-textarea': true }} name={this.name} placeholder={this.placeholder} required={this.required} readOnly={this.readonly} disabled={this.disabled} rows={this.rows} maxLength={maxLen > 0 ? maxLen : undefined} onInput={(e: Event) => this.handleInput(e)} onFocus={() => this.handleFocus()} onBlur={() => this.handleBlur()} {...ariaAttrs}>{this.value}</textarea>
          {this.loading && <span class="bc-field-loading-indicator" />}
          {this.clearable && this.value && !this.disabled && !this.readonly && <button type="button" class="bc-field-clear-btn" onClick={() => this.handleClear()} tabIndex={-1}>&times;</button>}
        </div>
        <div class="bc-field-footer">
          {showError && <div class="bc-field-error" id={`${this.name}-error`} role="alert">{this.validationMessage}</div>}
          {showHint && <div class="bc-field-hint" id={`${this.name}-hint`}>{this.hint}</div>}
          {this.showCount && maxLen > 0 && <div class="bc-field-counter">{currentLength}/{maxLen}</div>}
        </div>
      </div>
    );
  }
}
