import { Component, Prop, Event, EventEmitter, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

@Component({
  tag: 'bc-field-integer',
  styleUrl: 'bc-field-integer.css',
  shadow: true,
})
export class BcFieldInteger {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: number = 0;
  @Prop() placeholder: string = '';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() min: number = 0;
  @Prop() max: number = 0;
  @Prop() step: number = 1;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private handleInput(e: Event) {
    const target = e.target as HTMLInputElement;
    const oldValue = this.value;
    this.value = Number(target.value);
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue });
  }

  render() {
    return (
      <div class="bc-field">
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}</label>}
        <input
          type="number"
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
