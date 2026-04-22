import { Component, Prop, Event, EventEmitter, State, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';

@Component({
  tag: 'bc-field-rating',
  styleUrl: 'bc-field-rating.css',
  shadow: true,
})
export class BcFieldRating {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: number = 0;
  @Prop() maxStars: number = 5;
  @Prop() disabled: boolean = false;

  @State() hoverValue: number = 0;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private handleClick(star: number) {
    if (this.disabled) return;
    const oldValue = this.value;
    this.value = star;
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue });
  }

  render() {
    const stars = Array.from({ length: this.maxStars }, (_, i) => i + 1);
    const display = this.hoverValue || this.value;
    return (
      <div class="bc-field">
        {this.label && <label class="bc-field-label">{this.label}</label>}
        <div class="bc-rating">
          {stars.map(star => (
            <button
              type="button"
              class={{'bc-star': true, 'filled': star <= display}}
              disabled={this.disabled}
              onClick={() => this.handleClick(star)}
              onMouseEnter={() => { if (!this.disabled) this.hoverValue = star; }}
              onMouseLeave={() => { this.hoverValue = 0; }}
            >
              {star <= display ? '\u2605' : '\u2606'}
            </button>
          ))}
        </div>
      </div>
    );
  }
}
