import { Component, Prop, Element, Watch, h } from '@stencil/core';
import * as echarts from 'echarts';

@Component({ tag: 'bc-chart-pie', styleUrl: 'bc-chart-pie.css', shadow: false })
export class BcChartPie {
  @Element() el!: HTMLElement;
  @Prop() data: string = '[]';
  @Prop() chartTitle: string = '';
  private chart: echarts.ECharts | null = null;

  componentDidLoad() { this.renderChart(); }
  @Watch('data') onDataChange() { this.renderChart(); }
  disconnectedCallback() { this.chart?.dispose(); }

  private renderChart() {
    const container = this.el.querySelector('.bc-echart') as HTMLElement;
    if (!container) return;
    if (!this.chart) this.chart = echarts.init(container);
    let parsed: Array<{name: string; value: number}> = [];
    try { parsed = JSON.parse(this.data); } catch {}
    this.chart.setOption({
      title: { text: this.chartTitle, left: 'center', textStyle: { fontSize: 14 } },
      tooltip: { trigger: 'item' },
      series: [{ type: 'pie', radius: ['40%', '70%'], data: parsed, emphasis: { itemStyle: { shadowBlur: 10 } } }],
    });
  }

  render() { return (<div class="bc-chart-wrap"><div class="bc-echart"></div></div>); }
}
