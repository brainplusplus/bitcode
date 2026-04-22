import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-phone',
  styleUrl: 'bc-widget-phone.css',
  shadow: true,
})
export class BcWidgetPhone {@Prop() value: string = '';

  render() { return (<a href={`tel:${this.value}`} class="bc-link">{this.value}</a>); }
}
