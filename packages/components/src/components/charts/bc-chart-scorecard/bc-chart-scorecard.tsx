import { Component, Prop, Element, Watch, Method, Event, EventEmitter, h } from '@stencil/core';
import { ChartClickEvent } from '../../../core/types';
import * as echarts from 'echarts';

@Component({ tag: 'bc-chart-scorecard', styleUrl: 'bc-chart-scorecard.css', shadow: false })
export class BcChartScorecard {
  @Element() el!: HTMLElement;
  @Prop({ mutable: true }) value: string = '0';
  @Prop() height: string = '200px';
  @Event() lcChartClick!: EventEmitter<ChartClickEvent>;
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

  @Method() async updateData(newData: unknown): Promise<void> { this.value = String(newData); }
  @Method() async refresh(): Promise<void> { this.renderChart(); }
  @Method() async resize(): Promise<void> { this.chart?.resize(); }
  @Method() async exportImage(format: string = 'png'): Promise<string> { return this.chart?.getDataURL({ type: format as 'png' | 'jpeg' | 'svg', pixelRatio: 2 }) || ''; }

  render() { return (<div class="bc-chart-wrap"><div class="bc-echart" style={{ height: this.height }}></div></div>); }
}
