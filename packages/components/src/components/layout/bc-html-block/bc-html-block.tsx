import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-html-block',
  styleUrl: 'bc-html-block.css',
  shadow: true,
})
export class BcHtmlBlock {
  @Prop() content: string = '';

  render() {
    return (
      <div class="bc-html-block" innerHTML={this.content}></div>
    );
  }
}
