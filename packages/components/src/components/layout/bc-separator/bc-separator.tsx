import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-separator',
  styleUrl: 'bc-separator.css',
  shadow: true,
})
export class BcSeparator {
  @Prop() label: string = '';

  render() {
    return (
      <div class="bc-separator">
        <hr />
        {this.label && <span class="bc-separator-label">{this.label}</span>}
      </div>
    );
  }
}
