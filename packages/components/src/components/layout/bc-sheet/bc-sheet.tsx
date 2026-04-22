import { Component, h } from '@stencil/core';

@Component({
  tag: 'bc-sheet',
  styleUrl: 'bc-sheet.css',
  shadow: true,
})
export class BcSheet {
  render() {
    return (
      <div class="bc-sheet">
        <slot></slot>
      </div>
    );
  }
}
