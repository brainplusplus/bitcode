import { Component, Prop, Event, EventEmitter, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

@Component({
  tag: 'bc-field-select',
  styleUrl: 'bc-field-select.css',
  shadow: true,
})
export class BcFieldSelect {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = 'Select...';
  @Prop() options: string = '[]';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private getOptions(): string[] {
    try { return JSON.parse(this.options); } catch { return []; }
  }

  private handleChange(e: Event) {
    const target = e.target as HTMLSelectElement;
    const oldValue = this.value;
    this.value = target.value;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue });
  }

  render() {
    const opts = this.getOptions();
    return (
      <div class="bc-field">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        <select
          class="bc-field-input"
          disabled={this.disabled}
          required={this.required}
          onChange={(e) => this.handleChange(e)}
        >
          <option value="" disabled selected={!this.value}>{this.placeholder}</option>
          {opts.map(opt => <option value={opt} selected={opt === this.value}>{opt}</option>)}
        </select>
      </div>
    );
  }
}
