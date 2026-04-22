import { Component, Prop, h } from '@stencil/core';

@Component({
  tag: 'bc-widget-progress',
  styleUrl: 'bc-widget-progress.css',
  shadow: true,
})
export class BcWidgetProgress {
  @Prop() value: number = 0;
  @Prop() max: number = 100;
  @Prop() variant: string = 'primary';

  render() { const pct = Math.min(100, Math.max(0, (this.value / this.max) * 100)); return (<div class="bc-progress"><div class={`bc-progress-bar bc-progress-${this.variant}`} style={{width: pct + '%'}}></div><span class="bc-progress-text">{Math.round(pct)}%</span></div>); }
}
