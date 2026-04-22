import { Component, Prop, Element, Watch, h } from '@stencil/core';
import * as echarts from 'echarts';

@Component({ tag: 'bc-chart-gauge', styleUrl: 'bc-chart-gauge.css', shadow: false })
export class BcChartGauge {
  @Element() el!: HTMLElement;
  @Prop() value: string = '0';
  @Prop() max: string = '100';
  @Prop() chartTitle: string = '';
  private chart: echarts.ECharts | null = null;

  componentDidLoad() { this.renderChart(); }
  @Watch('value') onValueChange() { this.renderChart(); }
  disconnectedCallback() { this.chart?.dispose(); }

  private renderChart() {
    const container = this.el.querySelector('.bc-echart') as HTMLElement;
    if (!container) return;
    if (!this.chart) this.chart = echarts.init(container);
    this.chart.setOption({
      title: { text: this.chartTitle, left: 'center', textStyle: { fontSize: 14 } },
      series: [{ type: 'gauge', max: Number(this.max), data: [{ value: Number(this.value), name: this.chartTitle }], detail: { formatter: '{value}%' } }],
    });
  }

  render() { return (<div class="bc-chart-wrap"><div class="bc-echart"></div></div>); }
}
