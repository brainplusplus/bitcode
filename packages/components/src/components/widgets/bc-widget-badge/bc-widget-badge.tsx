import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-badge',
  styleUrl: 'bc-widget-badge.css',
  shadow: true,
})
export class BcWidgetBadge {
  @Prop() value: string = '';
  @Prop() variant: string = 'default';

  render() {
    return (<span class={`bc-badge bc-badge-${this.variant}`}>{this.value}</span>);
  }
}
