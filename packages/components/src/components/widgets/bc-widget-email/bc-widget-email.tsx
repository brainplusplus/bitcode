import { Component, Prop, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-email',
  styleUrl: 'bc-widget-email.css',
  shadow: false,
})
export class BcWidgetEmail {@Prop({ mutable: true }) value: string = '';

  @Method() async setValue(value: unknown): Promise<void> { this.value = String(value ?? ''); }
  @Method() async getValue(): Promise<unknown> { return this.value; }

  render() { return (<a href={`mailto:${this.value}`} class="bc-link">{this.value}</a>); }
}




