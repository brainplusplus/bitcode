import { Component, Prop, Event, EventEmitter, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

@Component({
  tag: 'bc-field-time',
  styleUrl: 'bc-field-time.css',
  shadow: true,
})
export class BcFieldTime {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = '';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private handleInput(e: Event) {
    const target = e.target as HTMLInputElement;
    const oldValue = this.value;
    this.value = target.value;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue });
  }

  render() {
    return (
      <div class="bc-field">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        <input
          type="time"
          class="bc-field-input"
          value={this.value}
          placeholder={this.placeholder}
          required={this.required}
          readOnly={this.readonly}
          disabled={this.disabled}
          onInput={(e: Event) => this.handleInput(e)}
        />
      </div>
    );
  }
}
