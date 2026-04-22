import { Component, Prop, h, State } from '@stencil/core';

@Component({
  tag: 'bc-widget-copy',
  styleUrl: 'bc-widget-copy.css',
  shadow: true,
})
export class BcWidgetCopy {
  @Prop() value: string = '';
  @State() copied: boolean = false;

  private async handleCopy() { try { await navigator.clipboard.writeText(this.value); this.copied = true; setTimeout(() => { this.copied = false; }, 2000); } catch {} }
  render() { return (<button type="button" class="bc-copy" onClick={() => this.handleCopy()}>{this.copied ? '\u2713 Copied' : '\u2398 Copy'}</button>); }
}
