import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getFieldClasses, validateFieldValue } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';
import { Editor } from '@tiptap/core';
import StarterKit from '@tiptap/starter-kit';

@Component({ tag: 'bc-field-richtext', styleUrl: 'bc-field-richtext.css', shadow: false })
export class BcFieldRichtext {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = 'Start typing...';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() toolbar: string = 'full';

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
  @Prop() defaultValue: string = '';
  @Prop() validateOn: ValidateOn | '' = '';

  @State() private _fieldState: FieldState = createFieldState('');
  private editor: Editor | null = null;
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

  componentDidLoad() {
    const el = this.el.querySelector('.tiptap-editor') as HTMLElement;
    if (!el) return;
    this.editor = new Editor({
      element: el,
      extensions: [StarterKit],
      content: this.value,
      editable: !this.readonly && !this.disabled,
      onUpdate: ({ editor }) => {
        const old = this.value;
        this.value = editor.getHTML();
        this._fieldState = markDirty(this._fieldState, this.value);
        this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
        if (this._getValidateOn() === 'change') this._runValidation();
      },
      onFocus: () => { this.lcFieldFocus.emit({ name: this.name, value: this.value }); },
      onBlur: () => {
        this._fieldState = markTouched(this._fieldState);
        this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true });
        if (this._getValidateOn() === 'blur') this._runValidation();
      },
    });
  }

  disconnectedCallback() { this.editor?.destroy(); }
  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }

  private cmd(c: string) {
    if (!this.editor) return;
    const chain = this.editor.chain().focus();
    switch(c) {
      case 'bold': chain.toggleBold().run(); break;
      case 'italic': chain.toggleItalic().run(); break;
      case 'strike': chain.toggleStrike().run(); break;
      case 'bulletList': chain.toggleBulletList().run(); break;
      case 'orderedList': chain.toggleOrderedList().run(); break;
      case 'blockquote': chain.toggleBlockquote().run(); break;
      case 'codeBlock': chain.toggleCodeBlock().run(); break;
      case 'undo': chain.undo().run(); break;
      case 'redo': chain.redo().run(); break;
    }
  }

  private handleClear() { const old = this.value; this.value = ''; this._fieldState = markDirty(this._fieldState, ''); this.editor?.commands.setContent(''); this.lcFieldClear.emit({ name: this.name, oldValue: old }); this.lcFieldChange.emit({ name: this.name, value: '', oldValue: old }); }

  private async _runValidation(): Promise<ValidationResult> {
    this.validationStatus = 'validating';
    const result = await validateFieldValue(this.value, { required: this.required, minLength: this.minLength ? this.minLength : undefined, maxLength: this.maxLength ? this.maxLength : undefined }, { validators: this.validators, customValidator: this.customValidator, serverValidator: this.serverValidator });
    if (result.valid) { this.validationStatus = 'valid'; this.validationMessage = ''; this.lcFieldValid.emit({ name: this.name, value: this.value }); }
    else { this.validationStatus = 'invalid'; this.validationMessage = result.errors[0] || ''; this.lcFieldInvalid.emit({ name: this.name, value: this.value, errors: result.errors }); }
    return result;
  }

  @Method() async validate(): Promise<ValidationResult> { return this._runValidation(); }
  @Method() async reset(): Promise<void> { this.value = this._fieldState.initialValue as string || this.defaultValue || ''; this._fieldState = createFieldState(this.value); this.validationStatus = 'none'; this.validationMessage = ''; this.editor?.commands.setContent(this.value); }
  @Method() async clear(): Promise<void> { this.handleClear(); }
  @Method() async setValue(value: string, emit: boolean = true): Promise<void> { const old = this.value; this.value = value; this._fieldState = markDirty(this._fieldState, value); this.editor?.commands.setContent(value); if (emit) this.lcFieldChange.emit({ name: this.name, value, oldValue: old }); }
  @Method() async getValue(): Promise<string> { return this.value; }
  @Method() async focusField(): Promise<void> { this.editor?.commands.focus(); }
  @Method() async blurField(): Promise<void> { this.editor?.commands.blur(); }
  @Method() async isDirty(): Promise<boolean> { return this._fieldState.dirty; }
  @Method() async isTouched(): Promise<boolean> { return this._fieldState.touched; }
  @Method() async setError(message: string): Promise<void> { this.validationStatus = 'invalid'; this.validationMessage = message; }
  @Method() async clearError(): Promise<void> { this.validationStatus = 'none'; this.validationMessage = ''; }

  render() {
    const fieldClasses = getFieldClasses({ size: this.size, validationStatus: this.validationStatus, disabled: this.disabled, readonly: this.readonly, loading: this.loading, dirty: this._fieldState.dirty, touched: this._fieldState.touched });
    const showError = this.validationStatus === 'invalid' && this.validationMessage;
    const showHint = this.hint && !showError;
    const full = this.toolbar === 'full';
    const textLen = this.editor?.getText().length || (this.value || '').replace(/<[^>]*>/g, '').length;

    return (
      <div class={{ ...fieldClasses, 'bc-richtext-wrap': true }}>
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}{this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}</label>}
        {!this.readonly && !this.disabled && (
          <div class="bc-rt-toolbar">
            <button type="button" onClick={() => this.cmd('bold')}><b>B</b></button>
            <button type="button" onClick={() => this.cmd('italic')}><i>I</i></button>
            <button type="button" onClick={() => this.cmd('strike')}><s>S</s></button>
            {full && <button type="button" onClick={() => this.cmd('bulletList')}>{'\u2022'}</button>}
            {full && <button type="button" onClick={() => this.cmd('orderedList')}>1.</button>}
            {full && <button type="button" onClick={() => this.cmd('blockquote')}>"</button>}
            {full && <button type="button" onClick={() => this.cmd('codeBlock')}>&lt;/&gt;</button>}
            <button type="button" onClick={() => this.cmd('undo')}>{'\u21A9'}</button>
            <button type="button" onClick={() => this.cmd('redo')}>{'\u21AA'}</button>
            {this.clearable && this.value && <button type="button" class="bc-field-clear-btn" onClick={() => this.handleClear()}>&times;</button>}
          </div>
        )}
        <div class="tiptap-editor"></div>
        <div class="bc-field-footer">
          {showError && <div class="bc-field-error" role="alert">{this.validationMessage}</div>}
          {showHint && <div class="bc-field-hint">{this.hint}</div>}
          {this.showCount && this.maxLength > 0 && <div class="bc-field-counter">{textLen}/{this.maxLength}</div>}
        </div>
      </div>
    );
  }
}
