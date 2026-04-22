import { Component, Prop, Event, EventEmitter, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

@Component({
  tag: 'bc-field-radio',
  styleUrl: 'bc-field-radio.css',
  shadow: true,
})
export class BcFieldRadio {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() options: string = '[]';
  @Prop() direction: string = 'vertical';
  @Prop() disabled: boolean = false;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private getOptions(): string[] {
    try { return JSON.parse(this.options); } catch { return []; }
  }

  private handleChange(opt: string) {
    if (this.disabled) return;
    const oldValue = this.value;
    this.value = opt;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue });
  }

  render() {
    const opts = this.getOptions();
    return (
      <div class="bc-field">
        {this.label && <label class="bc-field-label">{this.label}</label>}
        <div class={{'bc-radio-group': true, 'horizontal': this.direction === 'horizontal'}}>
          {opts.map(opt => (
            <label class="bc-radio-option">
              <input type="radio" name={this.name} value={opt} checked={opt === this.value} disabled={this.disabled} onChange={() => this.handleChange(opt)} />
              <span>{opt}</span>
            </label>
          ))}
        </div>
      </div>
    );
  }
}
