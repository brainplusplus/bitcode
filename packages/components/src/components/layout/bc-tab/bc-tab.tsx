import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-tab',
  styleUrl: 'bc-tab.css',
  shadow: true,
})
export class BcTab {
  @Prop() label: string = '';

  render() {
    return (
      <div class="bc-tab" role="tabpanel">
        <slot></slot>
      </div>
    );
  }
}
