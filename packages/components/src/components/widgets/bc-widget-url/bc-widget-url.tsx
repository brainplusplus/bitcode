import { Component, Prop, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-url',
  styleUrl: 'bc-widget-url.css',
  shadow: false,
})
export class BcWidgetUrl {@Prop({ mutable: true }) value: string = '';

  @Method() async setValue(value: unknown): Promise<void> { this.value = String(value ?? ''); }
  @Method() async getValue(): Promise<unknown> { return this.value; }

  render() { return (<a href={this.value} target="_blank" rel="noopener noreferrer" class="bc-link">{this.value} \u2197</a>); }
}




