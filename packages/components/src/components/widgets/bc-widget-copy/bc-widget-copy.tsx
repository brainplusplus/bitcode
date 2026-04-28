import { Component, Prop, State, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-copy',
  styleUrl: 'bc-widget-copy.css',
  shadow: false,
})
export class BcWidgetCopy {
  @Prop({ mutable: true }) value: string = '';
  @State() copied: boolean = false;

  private async handleCopy() {
    try {
      await navigator.clipboard.writeText(this.value);
      this.copied = true;
      setTimeout(() => { this.copied = false; }, 2000);
    } catch { /* clipboard API may not be available */ }
  }

  @Method() async setValue(value: unknown): Promise<void> { this.value = String(value ?? ''); }
  @Method() async getValue(): Promise<string> { return this.value; }

  render() {
    return (
      <button type="button" class="bc-copy" onClick={() => this.handleCopy()}>
        {this.copied ? '\u2713 Copied' : '\u2398 Copy'}
      </button>
    );
  }
}
