import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-email',
  styleUrl: 'bc-widget-email.css',
  shadow: true,
})
export class BcWidgetEmail {@Prop() value: string = '';

  render() { return (<a href={`mailto:${this.value}`} class="bc-link">{this.value}</a>); }
}
