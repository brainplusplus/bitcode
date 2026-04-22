import { Component, Prop, h } from '@stencil/core';

@Component({ tag: 'bc-chart-kpi', styleUrl: 'bc-chart-kpi.css', shadow: false })
export class BcChartKpi {
  @Prop() value: string = '0';
  @Prop() label: string = '';
  @Prop() trend: string = '';
  @Prop() valuePrefix: string = '';
  @Prop() valueSuffix: string = '';
  @Prop() color: string = 'primary';

  render() {
    const trendNum = Number(this.trend);
    const isUp = trendNum > 0;
    const isDown = trendNum < 0;
    return (
      <div class={'bc-kpi bc-kpi-' + this.color}>
        <div class="bc-kpi-value">
          {this.valuePrefix && <span class="bc-kpi-prefix">{this.valuePrefix}</span>}
          <span class="bc-kpi-number">{Number(this.value).toLocaleString()}</span>
          {this.valueSuffix && <span class="bc-kpi-suffix">{this.valueSuffix}</span>}
        </div>
        <div class="bc-kpi-label">{this.label}</div>
        {this.trend && (
          <div class={{'bc-kpi-trend': true, 'up': isUp, 'down': isDown}}>
            <span class="bc-kpi-trend-icon">{isUp ? '\u25B2' : isDown ? '\u25BC' : '\u25CF'}</span>
            <span>{Math.abs(trendNum)}%</span>
          </div>
        )}
      </div>
    );
  }
}
