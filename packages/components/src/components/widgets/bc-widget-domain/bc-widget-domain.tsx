import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-domain',
  styleUrl: 'bc-widget-domain.css',
  shadow: true,
})
export class BcWidgetDomain {
  @Prop() value: string = '[]';

  render() { return (<div class="bc-domain"><span class="bc-domain-label">Domain filter builder</span><pre class="bc-domain-value">{this.value}</pre></div>); }
}
