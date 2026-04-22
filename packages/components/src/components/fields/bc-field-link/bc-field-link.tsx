import { Component, Prop, Event, EventEmitter, State, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';
import { getApiClient } from '../../../core/api-client';

@Component({
  tag: 'bc-field-link',
  styleUrl: 'bc-field-link.css',
  shadow: false,
})
export class BcFieldLink {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = 'Search...';
  @Prop() model: string = '';
  @Prop() displayField: string = 'name';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() options: string = '[]';

  @Prop() lookupColumns: string = '[]';

  @State() query: string = '';
  @State() results: Array<Record<string, unknown>> = [];
  @State() showDropdown: boolean = false;
  @State() displayValue: string = '';
  @State() showLookup: boolean = false;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private debounceTimer: ReturnType<typeof setTimeout> | null = null;

  componentWillLoad() {
    this.displayValue = this.value;
  }

  private async search(q: string) {
    this.query = q;
    if (this.debounceTimer) clearTimeout(this.debounceTimer);
    if (q.length < 1) { this.results = []; this.showDropdown = false; return; }
    this.debounceTimer = setTimeout(async () => {
      try {
        const api = getApiClient();
        const items = await api.search(this.model, q);
        this.results = items;
        this.showDropdown = items.length > 0;
      } catch {
        this.results = [];
        this.showDropdown = false;
      }
    }, 300);
  }

  private select(item: Record<string, unknown>) {
    const old = this.value;
    this.value = String(item['id'] || '');
    this.displayValue = String(item[this.displayField] || item['name'] || item['id'] || '');
    this.query = '';
    this.showDropdown = false;
    this.results = [];
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  private clear() {
    const old = this.value;
    this.value = '';
    this.displayValue = '';
    this.lcFieldChange.emit({ name: this.name, value: '', oldValue: old });
  }

  private handleLookupSelect(e: CustomEvent) {
    const records = e.detail.records;
    if (records && records.length > 0) {
      this.select(records[0]);
    }
    this.showLookup = false;
  }

  render() {
    return (
      <div class="bc-field bc-link-wrap">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        <div class="bc-link-input-wrap">
          {this.value ? (
            <div class="bc-link-selected">
              <span class="bc-link-display">{this.displayValue}</span>
              {!this.readonly && !this.disabled && <button type="button" class="bc-link-clear" onClick={() => this.clear()}>{'\u00D7'}</button>}
            </div>
          ) : (
            <div class="bc-link-input-row">
              <input type="text" class="bc-field-input" placeholder={this.placeholder} readOnly={this.readonly} disabled={this.disabled} value={this.query} onInput={(e: Event) => this.search((e.target as HTMLInputElement).value)} onFocus={() => { if (this.results.length > 0) this.showDropdown = true; }} onBlur={() => setTimeout(() => { this.showDropdown = false; }, 200)} />
              {!this.readonly && !this.disabled && (
                <button type="button" class="bc-link-lookup-btn" onClick={() => { this.showLookup = true; }} title={'Browse ' + this.model}>
                  {'\uD83D\uDD0D'}
                </button>
              )}
            </div>
          )}
          {this.showDropdown && (
            <div class="bc-link-dropdown">
              {this.results.map(item => (
                <div class="bc-link-option" onMouseDown={() => this.select(item)}>
                  {String(item[this.displayField] || item['name'] || item['id'] || '')}
                </div>
              ))}
            </div>
          )}
        </div>
        <bc-lookup-modal
          open={this.showLookup}
          model={this.model}
          display-field={this.displayField}
          columns={this.lookupColumns}
          onLcLookupSelect={(e: CustomEvent) => this.handleLookupSelect(e)}
          onLcLookupClose={() => { this.showLookup = false; }}
        ></bc-lookup-modal>
      </div>
    );
  }
}
