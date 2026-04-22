import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-column',
  styleUrl: 'bc-column.css',
  shadow: true,
})
export class BcColumn {
  @Prop() width: number = 12;

  render() {
    const pct = (this.width / 12) * 100;
    return (
      <div class="bc-column" style={{ flex: `0 0 calc(${pct}% - var(--bc-spacing-md))`, maxWidth: `calc(${pct}% - var(--bc-spacing-md))` }}>
        <slot></slot>
      </div>
    );
  }
}
