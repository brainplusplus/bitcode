import { Component, Prop, Event, EventEmitter, State, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';
import { getApiClient } from '../../../core/api-client';

@Component({
  tag: 'bc-field-dynlink',
  styleUrl: 'bc-field-dynlink.css',
  shadow: false,
})
export class BcFieldDynlink {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = 'Search...';
  @Prop() modelfield: string = '';
  @Prop() model: string = '';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() options: string = '[]';

  @State() query: string = '';
  @State() results: Array<Record<string, unknown>> = [];
  @State() showDropdown: boolean = false;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private debounceTimer: ReturnType<typeof setTimeout> | null = null;

  private async search(q: string) {
    this.query = q;
    if (this.debounceTimer) clearTimeout(this.debounceTimer);
    if (q.length < 1 || !this.model) { this.results = []; this.showDropdown = false; return; }
    this.debounceTimer = setTimeout(async () => {
      try {
        const api = getApiClient();
        this.results = await api.search(this.model, q);
        this.showDropdown = this.results.length > 0;
      } catch { this.results = []; this.showDropdown = false; }
    }, 300);
  }

  private select(item: Record<string, unknown>) {
    const old = this.value;
    this.value = String(item['id'] || '');
    this.showDropdown = false;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  render() {
    return (
      <div class="bc-field bc-dynlink-wrap">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        <div class="bc-link-input-wrap">
          <input type="text" class="bc-field-input" placeholder={this.placeholder} readOnly={this.readonly} disabled={this.disabled} value={this.value || this.query} onInput={(e: Event) => this.search((e.target as HTMLInputElement).value)} onBlur={() => setTimeout(() => { this.showDropdown = false; }, 200)} />
          {this.showDropdown && (
            <div class="bc-link-dropdown">
              {this.results.map(item => (
                <div class="bc-link-option" onMouseDown={() => this.select(item)}>{String(item['name'] || item['id'] || '')}</div>
              ))}
            </div>
          )}
        </div>
      </div>
    );
  }
}
