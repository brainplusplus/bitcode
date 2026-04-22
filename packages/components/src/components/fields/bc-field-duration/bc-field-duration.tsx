import { Component, Prop, Event, EventEmitter, State, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

@Component({
  tag: 'bc-field-duration',
  styleUrl: 'bc-field-duration.css',
  shadow: true,
})
export class BcFieldDuration {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: number = 0;
  @Prop() disabled: boolean = false;

  @State() hours: number = 0;
  @State() minutes: number = 0;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  componentWillLoad() {
    this.hours = Math.floor(this.value / 3600);
    this.minutes = Math.floor((this.value % 3600) / 60);
  }

  private update() {
    const oldValue = this.value;
    this.value = this.hours * 3600 + this.minutes * 60;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue });
  }

  render() {
    return (
      <div class="bc-field">
        {this.label && <label class="bc-field-label">{this.label}</label>}
        <div class="bc-duration-inputs">
          <input type="number" class="bc-field-input bc-duration-part" value={this.hours} min={0} disabled={this.disabled} onInput={(e: Event) => { this.hours = Number((e.target as HTMLInputElement).value); this.update(); }} />
          <span class="bc-duration-sep">h</span>
          <input type="number" class="bc-field-input bc-duration-part" value={this.minutes} min={0} max={59} disabled={this.disabled} onInput={(e: Event) => { this.minutes = Number((e.target as HTMLInputElement).value); this.update(); }} />
          <span class="bc-duration-sep">m</span>
        </div>
      </div>
    );
  }
}
