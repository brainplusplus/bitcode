import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getFieldClasses, validateFieldValue } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';
import SignaturePad from 'signature_pad';

@Component({ tag: 'bc-field-signature', styleUrl: 'bc-field-signature.css', shadow: false })
export class BcFieldSignature {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() disabled: boolean = false;
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() width: number = 400;
  @Prop() height: number = 200;

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() tooltip: string = '';
  @Prop() defaultValue: string = '';
  @Prop() validateOn: ValidateOn | '' = '';

  @State() private _fieldState: FieldState = createFieldState('');
  private pad: SignaturePad | null = null;
  customValidator?: (value: unknown) => string | null | Promise<string | null>;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;
  @Event() lcFieldFocus!: EventEmitter<FieldFocusEvent>;
  @Event() lcFieldBlur!: EventEmitter<FieldBlurEvent>;
  @Event() lcFieldClear!: EventEmitter<FieldClearEvent>;
  @Event() lcFieldInvalid!: EventEmitter<FieldValidationEvent>;
  @Event() lcFieldValid!: EventEmitter<FieldValidEvent>;

  componentWillLoad() { this._fieldState = createFieldState(this.value || this.defaultValue); }

  componentDidLoad() {
    const canvas = this.el.querySelector('canvas') as HTMLCanvasElement;
    if (!canvas) return;
    canvas.width = this.width;
    canvas.height = this.height;
    this.pad = new SignaturePad(canvas, { backgroundColor: 'rgb(255,255,255)' });
    if (this.value) { this.pad.fromDataURL(this.value); }
    if (this.disabled || this.readonly) { this.pad.off(); }
    this.pad.addEventListener('beginStroke', () => {
      this.lcFieldFocus.emit({ name: this.name, value: this.value });
    });
    this.pad.addEventListener('endStroke', () => {
      const old = this.value;
      this.value = this.pad!.toDataURL();
      this._fieldState = markDirty(this._fieldState, this.value);
      this._fieldState = markTouched(this._fieldState);
      this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
      this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true });
      if (this._getValidateOn() === 'blur' || this._getValidateOn() === 'change') this._runValidation();
    });
  }

  disconnectedCallback() { this.pad?.off(); }
  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }

  private handleClear() {
    if (!this.pad) return;
    this.pad.clear();
    const old = this.value;
    this.value = '';
    this._fieldState = markDirty(this._fieldState, '');
    this.lcFieldClear.emit({ name: this.name, oldValue: old });
    this.lcFieldChange.emit({ name: this.name, value: '', oldValue: old });
  }

  private async _runValidation(): Promise<ValidationResult> {
    this.validationStatus = 'validating';
    const result = await validateFieldValue(this.value, { required: this.required }, { customValidator: this.customValidator });
    if (result.valid) { this.validationStatus = 'valid'; this.validationMessage = ''; this.lcFieldValid.emit({ name: this.name, value: this.value }); }
    else { this.validationStatus = 'invalid'; this.validationMessage = result.errors[0] || ''; this.lcFieldInvalid.emit({ name: this.name, value: this.value, errors: result.errors }); }
    return result;
  }

  @Method() async validate(): Promise<ValidationResult> { return this._runValidation(); }
  @Method() async reset(): Promise<void> { this.value = this._fieldState.initialValue as string || this.defaultValue || ''; this._fieldState = createFieldState(this.value); this.validationStatus = 'none'; this.validationMessage = ''; if (this.pad) { this.pad.clear(); if (this.value) this.pad.fromDataURL(this.value); } }
  @Method() async clear(): Promise<void> { this.handleClear(); }
  @Method() async setValue(value: string, emit: boolean = true): Promise<void> { const old = this.value; this.value = value; this._fieldState = markDirty(this._fieldState, value); if (this.pad) { this.pad.clear(); if (value) this.pad.fromDataURL(value); } if (emit) this.lcFieldChange.emit({ name: this.name, value, oldValue: old }); }
  @Method() async getValue(): Promise<string> { return this.value; }
  @Method() async focusField(): Promise<void> { (this.el.querySelector('canvas') as HTMLElement)?.focus(); }
  @Method() async blurField(): Promise<void> { (this.el.querySelector('canvas') as HTMLElement)?.blur(); }
  @Method() async isDirty(): Promise<boolean> { return this._fieldState.dirty; }
  @Method() async isTouched(): Promise<boolean> { return this._fieldState.touched; }
  @Method() async setError(message: string): Promise<void> { this.validationStatus = 'invalid'; this.validationMessage = message; }
  @Method() async clearError(): Promise<void> { this.validationStatus = 'none'; this.validationMessage = ''; }

  render() {
    const fieldClasses = getFieldClasses({ size: this.size, validationStatus: this.validationStatus, disabled: this.disabled, readonly: this.readonly, dirty: this._fieldState.dirty, touched: this._fieldState.touched });
    const showError = this.validationStatus === 'invalid' && this.validationMessage;
    const showHint = this.hint && !showError;

    return (
      <div class={{ ...fieldClasses, 'bc-sig-wrap': true }}>
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}{this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}</label>}
        <div class="bc-sig-canvas-wrap"><canvas></canvas></div>
        {!this.disabled && !this.readonly && <button type="button" class="bc-sig-clear" onClick={() => this.handleClear()}>Clear</button>}
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
