import { Component, Prop, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-badge',
  styleUrl: 'bc-widget-badge.css',
  shadow: false,
})
export class BcWidgetBadge {
  @Prop({ mutable: true }) value: string = '';
  @Prop() variant: string = 'default';

  @Method() async setValue(value: unknown): Promise<void> { this.value = String(value ?? ''); }
  @Method() async getValue(): Promise<unknown> { return this.value; }

  render() {
    return (<span class={`bc-badge bc-badge-${this.variant}`}>{this.value}</span>);
  }
}




