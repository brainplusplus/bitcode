import { Component, h } from '@stencil/core';
import { i18n } from '../../../core/i18n';

@Component({
  tag: 'bc-widget-handle',
  styleUrl: 'bc-widget-handle.css',
  shadow: true,
})
export class BcWidgetHandle {

  render() { return (<span class="bc-handle" title={i18n.t('handle.dragToReorder')}>{'☰'}</span>); }
}

