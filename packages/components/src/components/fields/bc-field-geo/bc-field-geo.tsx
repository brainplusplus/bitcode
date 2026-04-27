import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getFieldClasses, validateFieldValue } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';
import * as L from 'leaflet';

@Component({ tag: 'bc-field-geo', styleUrl: 'bc-field-geo.css', shadow: false })
export class BcFieldGeo {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() disabled: boolean = false;
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() placeholder: string = '';
  @Prop() drawMode: string = 'point';
  @Prop() zoom: number = 13;

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() clearable: boolean = false;
  @Prop() tooltip: string = '';
  @Prop() defaultValue: string = '';
  @Prop() validateOn: ValidateOn | '' = '';

  @State() private _fieldState: FieldState = createFieldState('');
  private map: L.Map | null = null;
  private marker: L.Marker | null = null;
  customValidator?: (value: unknown) => string | null | Promise<string | null>;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;
  @Event() lcFieldFocus!: EventEmitter<FieldFocusEvent>;
  @Event() lcFieldBlur!: EventEmitter<FieldBlurEvent>;
  @Event() lcFieldClear!: EventEmitter<FieldClearEvent>;
  @Event() lcFieldInvalid!: EventEmitter<FieldValidationEvent>;
  @Event() lcFieldValid!: EventEmitter<FieldValidEvent>;

  componentWillLoad() { this._fieldState = createFieldState(this.value || this.defaultValue); if (!this.value && this.defaultValue) this.value = this.defaultValue; }

  componentDidLoad() {
    const container = this.el.querySelector('.bc-map-container') as HTMLElement;
    if (!container) return;
    let lat = -6.2088, lng = 106.8456;
    if (this.value) {
      try { const p = JSON.parse(this.value); lat = p.lat || lat; lng = p.lng || lng; } catch {}
    }
    this.map = L.map(container).setView([lat, lng], this.zoom);
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', { attribution: '&copy; OpenStreetMap contributors' }).addTo(this.map);
    if (this.value) { this.marker = L.marker([lat, lng]).addTo(this.map); }
    if (!this.disabled && !this.readonly && this.drawMode === 'point') {
      this.map.on('click', (e: L.LeafletMouseEvent) => {
        const old = this.value;
        const newVal = JSON.stringify({ lat: e.latlng.lat, lng: e.latlng.lng });
        this.value = newVal;
        this._fieldState = markDirty(this._fieldState, newVal);
        if (this.marker) this.marker.setLatLng(e.latlng);
        else this.marker = L.marker(e.latlng).addTo(this.map!);
        this.lcFieldChange.emit({ name: this.name, value: newVal, oldValue: old });
      });
    }
  }

  disconnectedCallback() { this.map?.remove(); }
  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }

  private handleClear() {
    const old = this.value;
    this.value = '';
    this._fieldState = markDirty(this._fieldState, '');
    if (this.marker && this.map) { this.map.removeLayer(this.marker); this.marker = null; }
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
  @Method() async reset(): Promise<void> { this.value = this._fieldState.initialValue as string || this.defaultValue || ''; this._fieldState = createFieldState(this.value); this.validationStatus = 'none'; this.validationMessage = ''; }
  @Method() async clear(): Promise<void> { this.handleClear(); }
  @Method() async setValue(value: string, emit: boolean = true): Promise<void> { const old = this.value; this.value = value; this._fieldState = markDirty(this._fieldState, value); if (emit) this.lcFieldChange.emit({ name: this.name, value, oldValue: old }); }
  @Method() async getValue(): Promise<string> { return this.value; }
  @Method() async focusField(): Promise<void> { (this.el.querySelector('.bc-map-container') as HTMLElement)?.focus(); }
  @Method() async blurField(): Promise<void> { (this.el.querySelector('.bc-map-container') as HTMLElement)?.blur(); }
  @Method() async isDirty(): Promise<boolean> { return this._fieldState.dirty; }
  @Method() async isTouched(): Promise<boolean> { return this._fieldState.touched; }
  @Method() async setError(message: string): Promise<void> { this.validationStatus = 'invalid'; this.validationMessage = message; }
  @Method() async clearError(): Promise<void> { this.validationStatus = 'none'; this.validationMessage = ''; }

  render() {
    const fieldClasses = getFieldClasses({ size: this.size, validationStatus: this.validationStatus, disabled: this.disabled, readonly: this.readonly, dirty: this._fieldState.dirty, touched: this._fieldState.touched });
    const showError = this.validationStatus === 'invalid' && this.validationMessage;
    const showHint = this.hint && !showError;

    return (
      <div class={{ ...fieldClasses, 'bc-geo-wrap': true }}>
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}{this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}</label>}
        <div class="bc-map-container" tabIndex={0} onFocus={() => this.lcFieldFocus.emit({ name: this.name, value: this.value })} onBlur={() => { this._fieldState = markTouched(this._fieldState); this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true }); if (this._getValidateOn() === 'blur') this._runValidation(); }}></div>
        {this.value && <div class="bc-geo-coords">{this.value}</div>}
        {this.clearable && this.value && !this.disabled && !this.readonly && <button type="button" class="bc-field-clear-btn" onClick={() => this.handleClear()} tabIndex={-1}>&times; Clear</button>}
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
