import { Component, Prop, Element, Watch, Method, Event, EventEmitter, h } from '@stencil/core';
import { ChartClickEvent } from '../../../core/types';
import * as echarts from 'echarts';

@Component({ tag: 'bc-chart-gauge', styleUrl: 'bc-chart-gauge.css', shadow: false })
export class BcChartGauge {
  @Element() el!: HTMLElement;
  @Prop({ mutable: true }) value: string = '0';
  @Prop() max: string = '100';
  @Prop() chartTitle: string = '';
  @Prop() height: string = '300px';
  @Prop() loading: boolean = false;
  @Prop() animate: boolean = true;
  private chart: echarts.ECharts | null = null;
  @Event() lcChartClick!: EventEmitter<ChartClickEvent>;

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

  @Method() async updateData(newData: unknown): Promise<void> { this.value = String(newData); }
  @Method() async refresh(): Promise<void> { this.renderChart(); }
  @Method() async resize(): Promise<void> { this.chart?.resize(); }
  @Method() async exportImage(format: string = 'png'): Promise<string> { return this.chart?.getDataURL({ type: format as 'png' | 'jpeg' | 'svg', pixelRatio: 2 }) || ''; }

  render() { return (<div class="bc-chart-wrap"><div class="bc-echart" style={{ height: this.height }}></div></div>); }
}
