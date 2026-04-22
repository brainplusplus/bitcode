import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-url',
  styleUrl: 'bc-widget-url.css',
  shadow: true,
})
export class BcWidgetUrl {@Prop() value: string = '';

  render() { return (<a href={this.value} target="_blank" rel="noopener noreferrer" class="bc-link">{this.value} \u2197</a>); }
}
