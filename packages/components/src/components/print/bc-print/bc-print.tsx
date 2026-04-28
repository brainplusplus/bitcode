import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-print',
  styleUrl: 'bc-print.css',
  shadow: false,
})
export class BcPrint {
  @Prop() label: string = 'Print';
  @Prop() href: string = '';

  render() {
    return (
      <button type="button" class="bc-action-btn" onClick={() => { if (this.href) window.open(this.href); }}>
        {this.label}
      </button>
    );
  }
}

