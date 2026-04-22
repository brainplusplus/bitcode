import { Component, Prop, Element, Watch, h } from '@stencil/core';
import * as echarts from 'echarts';

@Component({ tag: 'bc-chart-scorecard', styleUrl: 'bc-chart-scorecard.css', shadow: false })
export class BcChartScorecard {
  @Element() el!: HTMLElement;
  @Prop() value: string = '0';
  @Prop() target: string = '100';
  @Prop() label: string = '';
  private chart: echarts.ECharts | null = null;

  componentDidLoad() { this.renderChart(); }
  @Watch('value') onValueChange() { this.renderChart(); }
  disconnectedCallback() { this.chart?.dispose(); }

  private renderChart() {
    const container = this.el.querySelector('.bc-echart') as HTMLElement;
    if (!container) return;
    if (!this.chart) this.chart = echarts.init(container);
    const val = Number(this.value);
    const tgt = Number(this.target);
    const pct = tgt > 0 ? Math.round((val / tgt) * 100) : 0;
    this.chart.setOption({
      series: [{ type: 'gauge', startAngle: 180, endAngle: 0, min: 0, max: tgt, data: [{ value: val, name: this.label }], detail: { formatter: pct + '%', fontSize: 20 }, title: { fontSize: 12 } }],
    });
  }

  render() { return (<div class="bc-chart-wrap"><div class="bc-echart"></div></div>); }
}
