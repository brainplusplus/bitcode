import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-report-link',
  styleUrl: 'bc-report-link.css',
  shadow: false,
})
export class BcReportLink {
  @Prop() label: string = 'View Report';
  @Prop() href: string = '';

  render() {
    return (
      <button type="button" class="bc-action-btn" onClick={() => { if (this.href) window.open(this.href); }}>
        {this.label}
      </button>
    );
  }
}

