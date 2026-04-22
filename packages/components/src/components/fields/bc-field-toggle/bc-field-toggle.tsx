import { Component, Prop, Event, EventEmitter, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

@Component({
  tag: 'bc-field-toggle',
  styleUrl: 'bc-field-toggle.css',
  shadow: true,
})
export class BcFieldToggle {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: boolean = false;
  @Prop() disabled: boolean = false;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private handleClick() {
    if (this.disabled) return;
    const oldValue = this.value;
    this.value = !this.value;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue });
  }

  render() {
    return (
      <div class="bc-field bc-field-inline">
        {this.label && <span class="bc-field-label">{this.label}</span>}
        <button type="button" class={{'bc-toggle': true, 'active': this.value}} disabled={this.disabled} onClick={() => this.handleClick()} aria-pressed={String(this.value)}>
          <span class="bc-toggle-knob"></span>
        </button>
      </div>
    );
  }
}
