import { Component, Prop, Event, EventEmitter, State, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';
import { getApiClient } from '../../../core/api-client';

@Component({
  tag: 'bc-field-tableselect',
  styleUrl: 'bc-field-tableselect.css',
  shadow: false,
})
export class BcFieldTableselect {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '[]';
  @Prop() placeholder: string = 'Search and add...';
  @Prop() model: string = '';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() options: string = '[]';

  @State() query: string = '';
  @State() results: Array<Record<string, unknown>> = [];
  @State() showDropdown: boolean = false;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private getValues(): string[] { try { return JSON.parse(this.value); } catch { return []; } }
  private debounceTimer: ReturnType<typeof setTimeout> | null = null;

  private async search(q: string) {
    this.query = q;
    if (this.debounceTimer) clearTimeout(this.debounceTimer);
    if (q.length < 1) { this.results = []; this.showDropdown = false; return; }
    this.debounceTimer = setTimeout(async () => {
      try {
        const api = getApiClient();
        this.results = await api.search(this.model, q);
        this.showDropdown = this.results.length > 0;
      } catch { this.results = []; this.showDropdown = false; }
    }, 300);
  }

  private addItem(val: string) {
    const items = this.getValues();
    if (items.includes(val)) return;
    const old = this.value;
    items.push(val);
    this.value = JSON.stringify(items);
    this.query = '';
    this.showDropdown = false;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  private removeItem(val: string) {
    const old = this.value;
    this.value = JSON.stringify(this.getValues().filter(v => v !== val));
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  render() {
    const items = this.getValues();
    return (
      <div class="bc-field bc-tableselect-wrap">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        {items.length > 0 && (
          <div class="bc-tableselect-list">
            {items.map(item => (
              <div class="bc-tableselect-item">
                <span>{item}</span>
                {!this.readonly && !this.disabled && <button type="button" class="bc-tableselect-remove" onClick={() => this.removeItem(item)}>&times;</button>}
              </div>
            ))}
          </div>
        )}
        {!this.readonly && !this.disabled && (
          <div class="bc-tableselect-search">
            <input type="text" class="bc-field-input" placeholder={this.placeholder} value={this.query} onInput={(e: Event) => this.search((e.target as HTMLInputElement).value)} onBlur={() => setTimeout(() => { this.showDropdown = false; }, 200)} />
            {this.showDropdown && (
              <div class="bc-tableselect-dropdown">
                {this.results.map(item => {
                  const val = String(item['name'] || item['id'] || '');
                  return <div class="bc-tableselect-option" onMouseDown={() => this.addItem(val)}>{val}</div>;
                })}
              </div>
            )}
          </div>
        )}
      </div>
    );
  }
}
