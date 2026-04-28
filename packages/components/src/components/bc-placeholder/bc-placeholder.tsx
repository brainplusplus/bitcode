import { Component, Prop, h } from '@stencil/core';
import { i18n } from '../../core/i18n';

@Component({
  tag: 'bc-placeholder',
  styleUrl: 'bc-placeholder.css',
  shadow: false,
})
export class BcPlaceholder {
  @Prop() text: string = '';

  render() {
    return (
      <div class="bc-placeholder">
        <span>{this.text || i18n.t('placeholder.default')}</span>
      </div>
    );
  }
}

