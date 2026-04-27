import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getFieldClasses } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';
import { EditorState } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import { basicSetup } from '@codemirror/basic-setup';
import { json } from '@codemirror/lang-json';

@Component({ tag: 'bc-field-json', styleUrl: 'bc-field-json.css', shadow: false })
export class BcFieldJson {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = '';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() language: string = '';
  @Prop() toolbar: string = 'full';

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() clearable: boolean = false;
  @Prop() tooltip: string = '';
  @Prop() loading: boolean = false;
  @Prop() defaultValue: string = '';
  @Prop() validateOn: ValidateOn | '' = '';

  @State() private _fieldState: FieldState = createFieldState('');
  private view: EditorView | null = null;
  customValidator?: (value: unknown) => string | null | Promise<string | null>;
  validators?: Array<{ rule: string | ((value: unknown) => boolean | Promise<boolean>); message: string }>;
  serverValidator?: string | ((value: unknown) => Promise<string | null>);

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;
  @Event() lcFieldFocus!: EventEmitter<FieldFocusEvent>;
  @Event() lcFieldBlur!: EventEmitter<FieldBlurEvent>;
  @Event() lcFieldClear!: EventEmitter<FieldClearEvent>;
  @Event() lcFieldInvalid!: EventEmitter<FieldValidationEvent>;
  @Event() lcFieldValid!: EventEmitter<FieldValidEvent>;

  componentWillLoad() { this._fieldState = createFieldState(this.value || this.defaultValue || '{}'); if (!this.value && this.defaultValue) this.value = this.defaultValue; }
  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }

  componentDidLoad() {
    const container = this.el.querySelector('.cm-json-container') as HTMLElement;
    if (!container) return;
    const state = EditorState.create({
      doc: this.value || '{}',
      extensions: [
        basicSetup,
        json(),
        EditorView.editable.of(!this.readonly && !this.disabled),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            const old = this.value;
            this.value = update.state.doc.toString();
            this._fieldState = markDirty(this._fieldState, this.value);
            this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
            if (this._getValidateOn() === 'change') this._runValidation();
          }
        }),
        EditorView.focusChangeEffect.of((_state, focusing) => {
          if (focusing) { this.lcFieldFocus.emit({ name: this.name, value: this.value }); }
          else { this._fieldState = markTouched(this._fieldState); this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true }); if (this._getValidateOn() === 'blur') this._runValidation(); }
          return null;
        }),
      ],
    });
    this.view = new EditorView({ state, parent: container });
  }

  disconnectedCallback() { this.view?.destroy(); }

  private handleClear() { const old = this.value; this.value = '{}'; this._fieldState = markDirty(this._fieldState, '{}'); if (this.view) { this.view.dispatch({ changes: { from: 0, to: this.view.state.doc.length, insert: '{}' } }); } this.lcFieldClear.emit({ name: this.name, oldValue: old }); this.lcFieldChange.emit({ name: this.name, value: '{}', oldValue: old }); }

  private async _runValidation(): Promise<ValidationResult> {
    this.validationStatus = 'validating';
    const errors: string[] = [];
    if (this.required && (!this.value || this.value === '{}')) { errors.push(BcSetup.getValidationMessage('required')); }
    if (this.value && this.value !== '{}') { try { JSON.parse(this.value); } catch { errors.push('Invalid JSON'); } }
    if (errors.length === 0 && this.customValidator) { const e = await this.customValidator(this.value); if (e) errors.push(e); }
    if (errors.length === 0 && this.serverValidator) {
      const { validateServer } = await import('../../../core/validation-engine');
      const r = await validateServer(this.value, this.serverValidator);
      if (!r.valid) errors.push(...r.errors);
    }
    if (errors.length === 0) { this.validationStatus = 'valid'; this.validationMessage = ''; this.lcFieldValid.emit({ name: this.name, value: this.value }); }
    else { this.validationStatus = 'invalid'; this.validationMessage = errors[0]; this.lcFieldInvalid.emit({ name: this.name, value: this.value, errors }); }
    return { valid: errors.length === 0, errors };
  }

  @Method() async validate(): Promise<ValidationResult> { return this._runValidation(); }
  @Method() async reset(): Promise<void> { this.value = this._fieldState.initialValue as string || this.defaultValue || '{}'; this._fieldState = createFieldState(this.value); this.validationStatus = 'none'; this.validationMessage = ''; if (this.view) { this.view.dispatch({ changes: { from: 0, to: this.view.state.doc.length, insert: this.value } }); } }
  @Method() async clear(): Promise<void> { this.handleClear(); }
  @Method() async setValue(value: string, emit: boolean = true): Promise<void> { const old = this.value; this.value = value; this._fieldState = markDirty(this._fieldState, value); if (this.view) { this.view.dispatch({ changes: { from: 0, to: this.view.state.doc.length, insert: value } }); } if (emit) this.lcFieldChange.emit({ name: this.name, value, oldValue: old }); }
  @Method() async getValue(): Promise<string> { return this.value; }
  @Method() async focusField(): Promise<void> { this.view?.focus(); }
  @Method() async blurField(): Promise<void> { this.view?.contentDOM.blur(); }
  @Method() async isDirty(): Promise<boolean> { return this._fieldState.dirty; }
  @Method() async isTouched(): Promise<boolean> { return this._fieldState.touched; }
  @Method() async setError(message: string): Promise<void> { this.validationStatus = 'invalid'; this.validationMessage = message; }
  @Method() async clearError(): Promise<void> { this.validationStatus = 'none'; this.validationMessage = ''; }

  render() {
    const fieldClasses = getFieldClasses({ size: this.size, validationStatus: this.validationStatus, disabled: this.disabled, readonly: this.readonly, loading: this.loading, dirty: this._fieldState.dirty, touched: this._fieldState.touched });
    const showError = this.validationStatus === 'invalid' && this.validationMessage;
    const showHint = this.hint && !showError;

    return (
      <div class={{ ...fieldClasses, 'bc-json-wrap': true }}>
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}{this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}</label>}
        {this.clearable && this.value && this.value !== '{}' && !this.disabled && !this.readonly && <button type="button" class="bc-field-clear-btn" onClick={() => this.handleClear()}>&times;</button>}
        <div class="cm-json-container"></div>
        <div class="bc-field-footer">
          {showError && <div class="bc-field-error" role="alert">{this.validationMessage}</div>}
          {showHint && <div class="bc-field-hint">{this.hint}</div>}
        </div>
      </div>
    );
  }
}
