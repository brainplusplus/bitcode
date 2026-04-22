import { Component, Prop, Event, EventEmitter, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

@Component({
  tag: 'bc-field-color',
  styleUrl: 'bc-field-color.css',
  shadow: true,
})
export class BcFieldColor {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '#000000';
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
        {this.label && <label class="bc-field-label">{this.label}</label>}
        <div class="bc-color-wrapper">
          <input type="color" value={this.value} disabled={this.disabled} onInput={(e: Event) => this.handleInput(e)} />
          <span class="bc-color-hex">{this.value}</span>
        </div>
      </div>
    );
  }
}
