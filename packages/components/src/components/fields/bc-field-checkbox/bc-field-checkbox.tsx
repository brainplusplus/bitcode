import { Component, Prop, Event, EventEmitter, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

@Component({
  tag: 'bc-field-checkbox',
  styleUrl: 'bc-field-checkbox.css',
  shadow: true,
})
export class BcFieldCheckbox {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: boolean = false;
  @Prop() disabled: boolean = false;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private handleChange(e: Event) {
    const target = e.target as HTMLInputElement;
    const oldValue = this.value;
    this.value = target.checked;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue });
  }

  render() {
    return (
      <div class="bc-field bc-field-inline">
        <label class="bc-checkbox-label">
          <input
            type="checkbox"
            checked={this.value}
            disabled={this.disabled}
            onChange={(e) => this.handleChange(e)}
          />
          <span class="bc-checkbox-text">{this.label}</span>
        </label>
      </div>
    );
  }
}
