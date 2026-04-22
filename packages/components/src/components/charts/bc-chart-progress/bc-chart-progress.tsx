import { Component, Prop, h } from '@stencil/core';

@Component({ tag: 'bc-chart-progress', styleUrl: 'bc-chart-progress.css', shadow: false })
export class BcChartProgress {
  @Prop() value: string = '0';
  @Prop() max: string = '100';
  @Prop() label: string = '';
  @Prop() color: string = 'primary';
  @Prop() showPercent: boolean = true;

  render() {
    const val = Number(this.value);
    const mx = Number(this.max);
    const pct = mx > 0 ? Math.min(100, Math.max(0, (val / mx) * 100)) : 0;
    return (
      <div class="bc-progress-chart">
        {this.label && <div class="bc-progress-label">{this.label}</div>}
        <div class="bc-progress-bar-wrap">
          <div class={'bc-progress-bar bc-progress-' + this.color} style={{ width: pct + '%' }}></div>
        </div>
        <div class="bc-progress-meta">
          <span>{val.toLocaleString()} / {mx.toLocaleString()}</span>
          {this.showPercent && <span class="bc-progress-pct">{Math.round(pct)}%</span>}
        </div>
      </div>
    );
  }
}
