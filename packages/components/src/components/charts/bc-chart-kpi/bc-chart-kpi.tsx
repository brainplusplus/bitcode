import { Component, Prop, Method, h } from '@stencil/core';

@Component({ tag: 'bc-chart-kpi', styleUrl: 'bc-chart-kpi.css', shadow: false })
export class BcChartKpi {
  @Prop({ mutable: true }) value: string = '0';
  @Prop() label: string = '';
  @Prop({ mutable: true }) trend: string = '';
  @Prop() valuePrefix: string = '';
  @Prop() valueSuffix: string = '';
  @Prop() color: string = 'primary';
  @Prop() loading: boolean = false;

  @Method() async updateData(newData: unknown): Promise<void> { if (typeof newData === 'object' && newData !== null) { const d = newData as Record<string, unknown>; if (d.value !== undefined) this.value = String(d.value); if (d.trend !== undefined) this.trend = String(d.trend); } else { this.value = String(newData); } }
  @Method() async refresh(): Promise<void> { /* no-op for static KPI */ }

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
