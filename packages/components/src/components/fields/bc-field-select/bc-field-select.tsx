import { Component, Prop, State, Event, EventEmitter, Method, Element, Watch, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn, OptionsLoadEvent, OptionsErrorEvent, OptionCreateEvent, FetchParams } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getAriaAttrs, getFieldClasses, validateFieldValue, debounce, findFormContainer } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';
import { fetchOptions } from '../../../core/data-fetcher';

interface SelectOption { label: string; value: string; group?: string; }

@Component({ tag: 'bc-field-select', styleUrl: 'bc-field-select.css', shadow: false })
export class BcFieldSelect {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = 'Select...';
  @Prop({ mutable: true }) options: string = '[]';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() clearable: boolean = false;
  @Prop() tooltip: string = '';
  @Prop({ mutable: true }) loading: boolean = false;
  @Prop() defaultValue: string = '';
  @Prop() validateOn: ValidateOn | '' = '';
  @Prop() dependOn: string = '';
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';

  @Prop() searchable: boolean = false;
  @Prop() serverSide: boolean = false;
  @Prop() multiple: boolean = false;
  @Prop() displayField: string = 'label';
  @Prop() valueField: string = 'value';
  @Prop() groupBy: string = '';
  @Prop() creatable: boolean = false;
  @Prop() pageSize: number = 50;
  @Prop() debounceMs: number = 300;
  @Prop() minSearchLength: number = 1;
  @Prop() noResultsText: string = 'No results';
  @Prop() loadingText: string = 'Loading...';
  @Prop() model: string = '';

  @State() private _fieldState: FieldState = createFieldState('');
  @State() private _isOpen: boolean = false;
  @State() private _searchQuery: string = '';
  @State() private _filteredOptions: SelectOption[] = [];
  @State() private _highlightIndex: number = -1;
  @State() private _loadedOptions: SelectOption[] = [];
  @State() private _fetchError: string = '';

  private _inputEl?: HTMLInputElement;
  private _dependListener?: (e: Event) => void;
  private _outsideClickListener?: (e: Event) => void;

  customValidator?: (value: unknown) => string | null | Promise<string | null>;
  validators?: Array<{ rule: string | ((value: unknown) => boolean | Promise<boolean>); message: string }>;
  serverValidator?: string | ((value: unknown) => Promise<string | null>);
  optionsFetcher?: (query: string, params: FetchParams) => Promise<unknown[]>;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;
  @Event() lcFieldFocus!: EventEmitter<FieldFocusEvent>;
  @Event() lcFieldBlur!: EventEmitter<FieldBlurEvent>;
  @Event() lcFieldClear!: EventEmitter<FieldClearEvent>;
  @Event() lcFieldInvalid!: EventEmitter<FieldValidationEvent>;
  @Event() lcFieldValid!: EventEmitter<FieldValidEvent>;
  @Event() lcOptionsLoad!: EventEmitter<OptionsLoadEvent>;
  @Event() lcOptionsError!: EventEmitter<OptionsErrorEvent>;
  @Event() lcOptionCreate!: EventEmitter<OptionCreateEvent>;
  @Event() lcDropdownOpen!: EventEmitter<void>;
  @Event() lcDropdownClose!: EventEmitter<void>;

  componentWillLoad() {
    this._fieldState = createFieldState(this.value || this.defaultValue);
    if (!this.value && this.defaultValue) this.value = this.defaultValue;
    this._parseOptions();
    if (this.dataSource && !this.serverSide) this._fetchInitialOptions();
  }

  componentDidLoad() {
    this._setupDependencyListener();
    this._outsideClickListener = (e: Event) => {
      if (this._isOpen && !this.el.contains(e.target as Node)) this._closeDropdown();
    };
    document.addEventListener('mousedown', this._outsideClickListener);
  }

  disconnectedCallback() {
    this._cleanupDependencyListener();
    if (this._outsideClickListener) document.removeEventListener('mousedown', this._outsideClickListener);
  }

  @Watch('options')
  onOptionsChange() { this._parseOptions(); }

  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }

  private _parseOptions() {
    try {
      const raw = JSON.parse(this.options);
      this._loadedOptions = (raw as unknown[]).map((opt: unknown) => {
        if (typeof opt === 'string') return { label: opt, value: opt };
        const o = opt as Record<string, unknown>;
        return { label: String(o[this.displayField] || o.label || o.name || o.text || ''), value: String(o[this.valueField] || o.value || o.id || ''), group: this.groupBy ? String(o[this.groupBy] || '') : undefined };
      });
    } catch { this._loadedOptions = []; }
    this._filteredOptions = [...this._loadedOptions];
  }

  private async _fetchInitialOptions() {
    this.loading = true;
    try {
      const result = await fetchOptions({ fetcher: this.optionsFetcher, element: this.el, dataSource: this.dataSource, localOptions: undefined, model: this.model, query: '', fetchHeaders: this.fetchHeaders, params: { pageSize: this.pageSize } });
      this._loadedOptions = (result as unknown[]).map((o: unknown) => {
        if (typeof o === 'string') return { label: o, value: o };
        const obj = o as Record<string, unknown>;
        return { label: String(obj[this.displayField] || obj.label || obj.name || ''), value: String(obj[this.valueField] || obj.value || obj.id || ''), group: this.groupBy ? String(obj[this.groupBy] || '') : undefined };
      });
      this._filteredOptions = [...this._loadedOptions];
      this.lcOptionsLoad.emit({ options: this._loadedOptions, total: this._loadedOptions.length });
    } catch (err) {
      this._fetchError = err instanceof Error ? err.message : 'Failed to load options';
      this.lcOptionsError.emit({ error: this._fetchError });
    } finally { this.loading = false; }
  }

  private _setupDependencyListener() {
    if (!this.dependOn) return;
    this._dependListener = (e: Event) => {
      const d = (e as CustomEvent<FieldChangeEvent>).detail;
      if (!d) return;
      const deps = this.dependOn.split(',').map(s => s.trim());
      if (!deps.includes(d.name)) return;
      const old = this.value;
      this.value = '';
      this._searchQuery = '';
      this._fieldState = createFieldState('');
      this._loadedOptions = [];
      this._filteredOptions = [];
      this.lcFieldChange.emit({ name: this.name, value: '', oldValue: old });
      if (this.dataSource) {
        const container = findFormContainer(this.el);
        const dependValues: Record<string, unknown> = {};
        deps.forEach(dep => {
          const field = container.querySelector(`[name="${dep}"]`) as HTMLElement & { value?: unknown };
          if (field) dependValues[dep] = field.value;
        });
        this._fetchDependentOptions(dependValues);
      }
    };
    document.addEventListener('lcFieldChange', this._dependListener);
  }

  private async _fetchDependentOptions(dependValues: Record<string, unknown>) {
    this.loading = true;
    try {
      const result = await fetchOptions({ fetcher: this.optionsFetcher, element: this.el, dataSource: this.dataSource, model: this.model, query: '', fetchHeaders: this.fetchHeaders, params: { dependValues, pageSize: this.pageSize } });
      this._loadedOptions = (result as unknown[]).map((o: unknown) => {
        if (typeof o === 'string') return { label: o, value: o };
        const obj = o as Record<string, unknown>;
        return { label: String(obj[this.displayField] || obj.label || obj.name || ''), value: String(obj[this.valueField] || obj.value || obj.id || '') };
      });
      this._filteredOptions = [...this._loadedOptions];
      this.lcOptionsLoad.emit({ options: this._loadedOptions, total: this._loadedOptions.length });
    } catch (err) {
      this._fetchError = err instanceof Error ? err.message : 'Failed to load options';
      this.lcOptionsError.emit({ error: this._fetchError });
    } finally { this.loading = false; }
  }

  private _cleanupDependencyListener() { if (this._dependListener) { document.removeEventListener('lcFieldChange', this._dependListener); this._dependListener = undefined; } }

  private _openDropdown() {
    if (this.disabled || this.readonly) return;
    this._isOpen = true;
    this._highlightIndex = -1;
    this._searchQuery = '';
    this._filteredOptions = [...this._loadedOptions];
    this.lcDropdownOpen.emit();
    this.lcFieldFocus.emit({ name: this.name, value: this.value });
    setTimeout(() => this._inputEl?.focus(), 10);
  }

  private _closeDropdown() {
    if (!this._isOpen) return;
    this._isOpen = false;
    this._searchQuery = '';
    this.lcDropdownClose.emit();
    this._fieldState = markTouched(this._fieldState);
    this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true });
    if (this._getValidateOn() === 'blur') this._runValidation();
  }

  private _handleSearch(q: string) {
    this._searchQuery = q;
    this._highlightIndex = -1;
    if (this.serverSide && (this.dataSource || this.optionsFetcher || this.model)) {
      if (q.length < this.minSearchLength) { this._filteredOptions = []; return; }
      debounce(`select-search-${this.name}`, () => this._fetchSearchResults(q), this.debounceMs);
    } else {
      const lower = q.toLowerCase();
      this._filteredOptions = this._loadedOptions.filter(o => o.label.toLowerCase().includes(lower) || o.value.toLowerCase().includes(lower));
    }
  }

  private async _fetchSearchResults(query: string) {
    this.loading = true;
    try {
      const result = await fetchOptions({ fetcher: this.optionsFetcher, element: this.el, dataSource: this.dataSource, model: this.model, query, fetchHeaders: this.fetchHeaders, params: { search: query, pageSize: this.pageSize } });
      this._filteredOptions = (result as unknown[]).map((o: unknown) => {
        if (typeof o === 'string') return { label: o, value: o };
        const obj = o as Record<string, unknown>;
        return { label: String(obj[this.displayField] || obj.label || obj.name || ''), value: String(obj[this.valueField] || obj.value || obj.id || '') };
      });
      this.lcOptionsLoad.emit({ options: this._filteredOptions, total: this._filteredOptions.length });
    } catch (err) {
      this._fetchError = err instanceof Error ? err.message : 'Search failed';
      this.lcOptionsError.emit({ error: this._fetchError });
    } finally { this.loading = false; }
  }

  private _selectOption(opt: SelectOption) {
    const old = this.value;
    this.value = opt.value;
    this._fieldState = markDirty(this._fieldState, this.value);
    this._closeDropdown();
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
    if (this._getValidateOn() === 'change') this._runValidation();
  }

  private _handleCreate() {
    if (!this.creatable || !this._searchQuery.trim()) return;
    const val = this._searchQuery.trim();
    const newOpt: SelectOption = { label: val, value: val };
    this._loadedOptions = [...this._loadedOptions, newOpt];
    this._selectOption(newOpt);
    this.lcOptionCreate.emit({ value: val });
  }

  private _handleKeyDown(e: KeyboardEvent) {
    if (!this._isOpen) { if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown') { e.preventDefault(); this._openDropdown(); } return; }
    switch (e.key) {
      case 'ArrowDown': e.preventDefault(); this._highlightIndex = Math.min(this._highlightIndex + 1, this._filteredOptions.length - 1); break;
      case 'ArrowUp': e.preventDefault(); this._highlightIndex = Math.max(this._highlightIndex - 1, 0); break;
      case 'Enter': e.preventDefault(); if (this._highlightIndex >= 0 && this._filteredOptions[this._highlightIndex]) this._selectOption(this._filteredOptions[this._highlightIndex]); else if (this.creatable) this._handleCreate(); break;
      case 'Escape': e.preventDefault(); this._closeDropdown(); break;
    }
  }

  private handleClear() { const old = this.value; this.value = ''; this._fieldState = markDirty(this._fieldState, ''); this.lcFieldClear.emit({ name: this.name, oldValue: old }); this.lcFieldChange.emit({ name: this.name, value: '', oldValue: old }); }

  private _getDisplayLabel(): string {
    if (!this.value) return '';
    const opt = this._loadedOptions.find(o => o.value === this.value);
    return opt ? opt.label : this.value;
  }

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
  @Method() async focusField(): Promise<void> { this._openDropdown(); }
  @Method() async blurField(): Promise<void> { this._closeDropdown(); }
  @Method() async isDirty(): Promise<boolean> { return this._fieldState.dirty; }
  @Method() async isTouched(): Promise<boolean> { return this._fieldState.touched; }
  @Method() async setError(message: string): Promise<void> { this.validationStatus = 'invalid'; this.validationMessage = message; }
  @Method() async clearError(): Promise<void> { this.validationStatus = 'none'; this.validationMessage = ''; }
  @Method() async setOptions(opts: unknown[]): Promise<void> { this.options = JSON.stringify(opts); }
  @Method() async loadOptions(query?: string): Promise<void> { if (query !== undefined) await this._fetchSearchResults(query); else await this._fetchInitialOptions(); }
  @Method() async reloadOptions(): Promise<void> { await this._fetchInitialOptions(); }
  @Method() async getOptions(): Promise<SelectOption[]> { return this._loadedOptions; }
  @Method() async getSelectedOptions(): Promise<SelectOption[]> { return this._loadedOptions.filter(o => o.value === this.value); }
  @Method() async open(): Promise<void> { this._openDropdown(); }
  @Method() async close(): Promise<void> { this._closeDropdown(); }

  render() {
    const fieldClasses = getFieldClasses({ size: this.size, validationStatus: this.validationStatus, disabled: this.disabled, readonly: this.readonly, loading: this.loading, dirty: this._fieldState.dirty, touched: this._fieldState.touched });
    const ariaAttrs = getAriaAttrs({ name: this.name, required: this.required, disabled: this.disabled, readonly: this.readonly, validationStatus: this.validationStatus, validationMessage: this.validationMessage, hint: this.hint });
    const showError = this.validationStatus === 'invalid' && this.validationMessage;
    const showHint = this.hint && !showError;
    const displayLabel = this._getDisplayLabel();

    return (
      <div class={fieldClasses}>
        {this.label && <label class="bc-field-label" htmlFor={this.name}>{this.label}{this.required && <span class="required">*</span>}{this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}</label>}

        <div class="bc-select-wrapper" onKeyDown={(e) => this._handleKeyDown(e)}>
          <button type="button" class={{ 'bc-select-trigger': true, 'bc-select-open': this._isOpen, 'bc-field-input': true, [`bc-field-input-${this.size}`]: this.size !== 'md' }} disabled={this.disabled || this.readonly} onClick={() => this._isOpen ? this._closeDropdown() : this._openDropdown()} {...ariaAttrs} role="combobox" aria-expanded={String(this._isOpen)} aria-haspopup="listbox">
            <span class={{ 'bc-select-value': true, 'bc-select-placeholder': !this.value }}>{displayLabel || this.placeholder}</span>
            <span class="bc-select-arrow">{this._isOpen ? '\u25B2' : '\u25BC'}</span>
          </button>

          {this.clearable && this.value && !this.disabled && !this.readonly && <button type="button" class="bc-field-clear-btn" onClick={(e) => { e.stopPropagation(); this.handleClear(); }} tabIndex={-1}>&times;</button>}
          {this.loading && <span class="bc-field-loading-indicator" />}

          {this._isOpen && (
            <div class="bc-select-dropdown" role="listbox">
              {this.searchable && (
                <div class="bc-select-search">
                  <input ref={(el) => this._inputEl = el} type="text" class="bc-select-search-input" placeholder="Search..." value={this._searchQuery} onInput={(e) => this._handleSearch((e.target as HTMLInputElement).value)} />
                </div>
              )}
              <div class="bc-select-options">
                {this.loading && <div class="bc-select-loading">{this.loadingText}</div>}
                {!this.loading && this._filteredOptions.length === 0 && (
                  <div class="bc-select-no-results">
                    {this._searchQuery ? this.noResultsText : 'No options available'}
                    {this.creatable && this._searchQuery && <button type="button" class="bc-select-create" onClick={() => this._handleCreate()}>Create "{this._searchQuery}"</button>}
                  </div>
                )}
                {!this.loading && this._filteredOptions.map((opt, i) => (
                  <div class={{ 'bc-select-option': true, 'bc-select-option-selected': opt.value === this.value, 'bc-select-option-highlighted': i === this._highlightIndex }} onMouseDown={() => this._selectOption(opt)} onMouseEnter={() => { this._highlightIndex = i; }} role="option" aria-selected={String(opt.value === this.value)}>
                    {opt.label}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        <div class="bc-field-footer">
          {showError && <div class="bc-field-error" id={`${this.name}-error`} role="alert">{this.validationMessage}</div>}
          {showHint && <div class="bc-field-hint" id={`${this.name}-hint`}>{this.hint}</div>}
        </div>
      </div>
    );
  }
}
