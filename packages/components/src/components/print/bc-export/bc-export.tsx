import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-export',
  styleUrl: 'bc-export.css',
  shadow: false,
})
export class BcExport {
  @Prop() label: string = 'Export';
  @Prop() href: string = '';

  render() {
    return (
      <button type="button" class="bc-action-btn" onClick={() => { if (this.href) window.open(this.href); }}>
        {this.label}
      </button>
    );
  }
}

