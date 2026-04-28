import { Component, Prop, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-domain',
  styleUrl: 'bc-widget-domain.css',
  shadow: false,
})
export class BcWidgetDomain {
  @Prop({ mutable: true }) value: string = '';

  @Method() async setValue(value: unknown): Promise<void> { this.value = String(value ?? ''); }
  @Method() async getValue(): Promise<unknown> { return this.value; }

  render() { return (<div class="bc-domain"><span class="bc-domain-label">Domain filter builder</span><pre class="bc-domain-value">{this.value}</pre></div>); }
}




