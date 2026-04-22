import { Component, Prop, Event, EventEmitter, State, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';
import { getApiClient } from '../../../core/api-client';

@Component({
  tag: 'bc-field-tags',
  styleUrl: 'bc-field-tags.css',
  shadow: false,
})
export class BcFieldTags {
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

  private addTag(val: string) {
    const tags = this.getValues();
    if (tags.includes(val)) return;
    const old = this.value;
    tags.push(val);
    this.value = JSON.stringify(tags);
    this.query = '';
    this.showDropdown = false;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  private removeTag(val: string) {
    const old = this.value;
    const tags = this.getValues().filter(t => t !== val);
    this.value = JSON.stringify(tags);
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  render() {
    const tags = this.getValues();
    return (
      <div class="bc-field bc-tags-wrap">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        <div class="bc-tags-container">
          {tags.map(tag => (
            <span class="bc-tag">
              {tag}
              {!this.readonly && !this.disabled && <button type="button" class="bc-tag-remove" onClick={() => this.removeTag(tag)}>&times;</button>}
            </span>
          ))}
          {!this.readonly && !this.disabled && (
            <div class="bc-tags-input-wrap">
              <input type="text" class="bc-tags-input" placeholder={tags.length === 0 ? this.placeholder : ''} value={this.query} onInput={(e: Event) => this.search((e.target as HTMLInputElement).value)} onBlur={() => setTimeout(() => { this.showDropdown = false; }, 200)} />
              {this.showDropdown && (
                <div class="bc-tags-dropdown">
                  {this.results.map(item => {
                    const val = String(item['name'] || item['id'] || '');
                    return <div class="bc-tags-option" onMouseDown={() => this.addTag(val)}>{val}</div>;
                  })}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    );
  }
}
