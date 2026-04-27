import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, getFieldClasses, validateFieldValue } from '../../../core/field-utils';
import { getApiClient } from '../../../core/api-client';

@Component({
  tag: 'bc-field-image',
  styleUrl: 'bc-field-image.css',
  shadow: false,
})
export class BcFieldImage {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() accept: string = 'image/*';
  @Prop() maxSize: string = '10MB';
  @Prop() required: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() preview: boolean = true;
  @Prop() showDownload: boolean = false;
  @Prop() model: string = '';
  @Prop() recordId: string = '';
  @Prop() fieldName: string = '';

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() tooltip: string = '';
  @Prop() loading: boolean = false;
  @Prop() defaultValue: string = '';
  @Prop() validateOn: ValidateOn | '' = '';

  @State() imageUrl: string = '';
  @State() imageName: string = '';
  @State() uploading: boolean = false;
  @State() uploadError: string = '';
  @State() isDragging: boolean = false;
  @State() private _fieldState: FieldState = createFieldState('');

  customValidator?: (value: unknown) => string | null | Promise<string | null>;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;
  @Event() lcFieldFocus!: EventEmitter<FieldFocusEvent>;
  @Event() lcFieldBlur!: EventEmitter<FieldBlurEvent>;
  @Event() lcFieldClear!: EventEmitter<FieldClearEvent>;
  @Event() lcFieldInvalid!: EventEmitter<FieldValidationEvent>;
  @Event() lcFieldValid!: EventEmitter<FieldValidEvent>;

  componentWillLoad() { this._fieldState = createFieldState(this.value || this.defaultValue); }

  private async _runValidation(): Promise<ValidationResult> {
    this.validationStatus = 'validating';
    const result = await validateFieldValue(this.value, { required: this.required }, { customValidator: this.customValidator });
    if (result.valid) { this.validationStatus = 'valid'; this.validationMessage = ''; this.lcFieldValid.emit({ name: this.name, value: this.value }); }
    else { this.validationStatus = 'invalid'; this.validationMessage = result.errors[0] || ''; this.lcFieldInvalid.emit({ name: this.name, value: this.value, errors: result.errors }); }
    return result;
  }

  @Method() async validate(): Promise<ValidationResult> { return this._runValidation(); }
  @Method() async reset(): Promise<void> { this.value = this._fieldState.initialValue as string || this.defaultValue || ''; this.imageUrl = ''; this.imageName = ''; this._fieldState = createFieldState(this.value); this.validationStatus = 'none'; this.validationMessage = ''; this.uploadError = ''; }
  @Method() async clear(): Promise<void> { this.removeImage(); }
  @Method() async setValue(value: string, emit: boolean = true): Promise<void> { const old = this.value; this.value = value; this._fieldState = markDirty(this._fieldState, value); if (emit) this.lcFieldChange.emit({ name: this.name, value, oldValue: old }); }
  @Method() async getValue(): Promise<string> { return this.value; }
  @Method() async focusField(): Promise<void> { (this.el.querySelector('.bc-image-dropzone') as HTMLElement)?.focus(); }
  @Method() async blurField(): Promise<void> { (document.activeElement as HTMLElement)?.blur(); }
  @Method() async isDirty(): Promise<boolean> { return this._fieldState.dirty; }
  @Method() async isTouched(): Promise<boolean> { return this._fieldState.touched; }
  @Method() async setError(message: string): Promise<void> { this.validationStatus = 'invalid'; this.validationMessage = message; }
  @Method() async clearError(): Promise<void> { this.validationStatus = 'none'; this.validationMessage = ''; this.uploadError = ''; }

  private fileInput!: HTMLInputElement;

  private parseMaxSize(): number {
    const str = this.maxSize.toUpperCase();
    const num = parseFloat(str);
    if (str.endsWith('MB')) return num * 1024 * 1024;
    if (str.endsWith('KB')) return num * 1024;
    if (str.endsWith('GB')) return num * 1024 * 1024 * 1024;
    return num;
  }

  private handleDragOver(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    this.isDragging = true;
  }

  private handleDragLeave(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    this.isDragging = false;
  }

  private handleDrop(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    this.isDragging = false;
    const file = e.dataTransfer?.files?.[0];
    if (file) this.processFile(file);
  }

  private handleInputChange(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (file) {
      this.processFile(file);
      input.value = '';
    }
  }

  private processFile(file: File) {
    if (!file.type.startsWith('image/')) {
      this.uploadError = 'Only image files are allowed';
      return;
    }

    const maxSize = this.parseMaxSize();
    if (file.size > maxSize) {
      this.uploadError = `File too large (max ${this.maxSize})`;
      return;
    }

    this.uploadError = '';
    this.uploadImage(file);
  }

  private async uploadImage(file: File) {
    this.uploading = true;
    this.uploadError = '';

    try {
      const form = new FormData();
      form.append('file', file);
      if (this.model) form.append('model', this.model);
      if (this.recordId) form.append('record_id', this.recordId);
      if (this.fieldName) form.append('field_name', this.fieldName);

      const client = getApiClient();
      const result = await client.uploadFile(form);

      const oldValue = this.value;
      this.value = result.id;
      this.imageUrl = result.thumbnail_url || result.url;
      this.imageName = result.name;
      this._fieldState = markDirty(this._fieldState, this.value);
      this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue });
    } catch (err) {
      this.uploadError = err instanceof Error ? err.message : 'Upload failed';
    } finally {
      this.uploading = false;
    }
  }

  private removeImage() {
    const oldValue = this.value;
    this.value = '';
    this.imageUrl = '';
    this.imageName = '';
    this._fieldState = markDirty(this._fieldState, '');
    this.lcFieldClear.emit({ name: this.name, oldValue });
    this.lcFieldChange.emit({ name: this.name, value: '', oldValue });
  }

  private handleDownload() {
    if (!this.imageUrl) return;
    const a = document.createElement('a');
    a.href = this.imageUrl;
    a.download = this.imageName || 'image';
    a.target = '_blank';
    a.click();
  }

  render() {
    const fieldClasses = getFieldClasses({ size: this.size, validationStatus: this.validationStatus, disabled: this.disabled, readonly: this.readonly, loading: this.loading, dirty: this._fieldState.dirty, touched: this._fieldState.touched });
    const showError = (this.validationStatus === 'invalid' && this.validationMessage) || this.uploadError;
    const showHint = this.hint && !showError;

    return (
      <div class={fieldClasses}>
        {this.label && (
          <label class="bc-field-label">
            {this.label}
            {this.required && <span class="required">*</span>}
            {this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}
          </label>
        )}

        {this.imageUrl && this.preview ? (
          <div class="bc-image-preview-wrapper">
            <div class="bc-image-preview">
              <img src={this.imageUrl} alt={this.imageName} class="bc-image-preview-img" />
              <div class="bc-image-preview-overlay">
                {!this.disabled && (
                  <button class="bc-image-overlay-btn" onClick={() => this.fileInput?.click()} title="Change image">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                    </svg>
                  </button>
                )}
                {this.showDownload && (
                  <button class="bc-image-overlay-btn" onClick={() => this.handleDownload()} title="Download">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                      <polyline points="7 10 12 15 17 10" />
                      <line x1="12" y1="15" x2="12" y2="3" />
                    </svg>
                  </button>
                )}
                {!this.disabled && (
                  <button class="bc-image-overlay-btn bc-image-overlay-btn-danger" onClick={() => this.removeImage()} title="Remove">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
                    </svg>
                  </button>
                )}
              </div>
            </div>
            {this.imageName && <span class="bc-image-name">{this.imageName}</span>}
          </div>
        ) : (
          <div
            class={{ 'bc-image-dropzone': true, 'dragging': this.isDragging, 'disabled': this.disabled }}
            onDragOver={(e) => this.handleDragOver(e)}
            onDragLeave={(e) => this.handleDragLeave(e)}
            onDrop={(e) => this.handleDrop(e)}
            onClick={() => !this.disabled && this.fileInput?.click()}
          >
            {this.uploading ? (
              <div class="bc-image-uploading">
                <div class="bc-image-spinner"></div>
                <span>Uploading...</span>
              </div>
            ) : (
              <div class="bc-image-dropzone-content">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                  <circle cx="8.5" cy="8.5" r="1.5" />
                  <polyline points="21 15 16 10 5 21" />
                </svg>
                <span class="bc-image-dropzone-text">Click or drag image to upload</span>
                <span class="bc-image-dropzone-hint">Max {this.maxSize}</span>
              </div>
            )}
          </div>
        )}

        <input
          type="file"
          ref={(el) => this.fileInput = el as HTMLInputElement}
          accept={this.accept}
          disabled={this.disabled}
          onChange={(e) => this.handleInputChange(e)}
          style={{ display: 'none' }}
        />

        <div class="bc-field-footer">
          {this.uploadError && <div class="bc-field-error" role="alert">{this.uploadError}</div>}
          {!this.uploadError && showError && <div class="bc-field-error" role="alert">{this.validationMessage}</div>}
          {showHint && <div class="bc-field-hint">{this.hint}</div>}
        </div>
      </div>
    );
  }
}
