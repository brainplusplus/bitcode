import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-row',
  styleUrl: 'bc-row.css',
  shadow: false,
})
export class BcRow {
  @Prop() gap: string = 'md';

  render() {
    return (
      <div class={`bc-row gap-${this.gap}`}>
        <slot></slot>
      </div>
    );
  }
}

