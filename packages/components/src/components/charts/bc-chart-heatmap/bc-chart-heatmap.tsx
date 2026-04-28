import { Component, Prop, Element, Watch, Method, Event, EventEmitter, h } from '@stencil/core';
import { ChartClickEvent } from '../../../core/types';
import * as echarts from 'echarts';

@Component({ tag: 'bc-chart-heatmap', styleUrl: 'bc-chart-heatmap.css', shadow: false })
export class BcChartHeatmap {
  @Element() el!: HTMLElement;
  @Prop({ mutable: true }) data: string = '[]';
  @Prop() chartTitle: string = '';
  @Prop() colors: string = '';
  @Prop() legend: boolean = true;
  @Prop() tooltipEnabled: boolean = true;
  @Prop() animate: boolean = true;
  @Prop() height: string = '300px';
  @Prop({ mutable: true }) loading: boolean = false;
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() refreshInterval: number = 0;
  @Event() lcChartClick!: EventEmitter<ChartClickEvent>;
  private chart: echarts.ECharts | null = null;

  componentDidLoad() { this.renderChart(); }
  @Watch('data') onDataChange() { this.renderChart(); }
  disconnectedCallback() { this.chart?.dispose(); }

  private renderChart() {
    const container = this.el.querySelector('.bc-echart') as HTMLElement;
    if (!container) return;
    if (!this.chart) this.chart = echarts.init(container);
    let parsed: Array<[number, number, number]> = [];
    try { parsed = JSON.parse(this.data); } catch {}
    this.chart.setOption({
      title: { text: this.chartTitle, left: 'center', textStyle: { fontSize: 14 } },
      tooltip: { position: 'top' },
      grid: { height: '50%', top: '10%' },
      visualMap: { min: 0, max: 10, calculable: true, orient: 'horizontal', left: 'center', bottom: '15%' },
      series: [{ type: 'heatmap', data: parsed, emphasis: { itemStyle: { shadowBlur: 10 } } }],
    });
  }  @Method() async updateData(newData: unknown): Promise<void> { this.data = typeof newData === 'string' ? newData : JSON.stringify(newData); }
  @Method() async setData(newData: unknown): Promise<void> { this.data = typeof newData === 'string' ? newData : JSON.stringify(newData); }
  @Method() async refresh(): Promise<void> { this.renderChart(); }
  @Method() async resize(): Promise<void> { this.chart?.resize(); }
  @Method() async exportImage(format: string = 'png'): Promise<string> { return this.chart?.getDataURL({ type: format as 'png' | 'jpeg' | 'svg', pixelRatio: 2 }) || ''; }



  render() { return (<div class="bc-chart-wrap"><div class="bc-echart"></div></div>); }
}


