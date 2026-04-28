import { Component, Prop, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-phone',
  styleUrl: 'bc-widget-phone.css',
  shadow: false,
})
export class BcWidgetPhone {@Prop({ mutable: true }) value: string = '';

  @Method() async setValue(value: unknown): Promise<void> { this.value = String(value ?? ''); }
  @Method() async getValue(): Promise<unknown> { return this.value; }

  render() { return (<a href={`tel:${this.value}`} class="bc-link">{this.value}</a>); }
}




