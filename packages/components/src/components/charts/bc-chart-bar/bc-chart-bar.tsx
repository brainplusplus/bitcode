import { Component, Prop, Element, Watch, Method, Event, EventEmitter, h } from '@stencil/core';
import { ChartClickEvent, DataFetcher } from '../../../core/types';
import { fetchData } from '../../../core/data-fetcher';
import * as echarts from 'echarts';

@Component({ tag: 'bc-chart-bar', styleUrl: 'bc-chart-bar.css', shadow: false })
export class BcChartBar {
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

  private chart: echarts.ECharts | null = null;
  private _refreshTimer: ReturnType<typeof setInterval> | null = null;
  dataFetcher?: DataFetcher;

  @Event() lcChartClick!: EventEmitter<ChartClickEvent>;

  componentDidLoad() {
    this.renderChart();
    if (this.dataSource || this.dataFetcher) this._fetchData();
    if (this.refreshInterval > 0) this._refreshTimer = setInterval(() => this._fetchData(), this.refreshInterval);
  }

  @Watch('data') onDataChange() { this.renderChart(); }
  disconnectedCallback() { this.chart?.dispose(); if (this._refreshTimer) clearInterval(this._refreshTimer); }

  private async _fetchData() {
    this.loading = true;
    try {
      const result = await fetchData({ fetcher: this.dataFetcher, element: this.el, dataSource: this.dataSource, localData: undefined, fetchHeaders: this.fetchHeaders });
      this.data = JSON.stringify(result.data);
    } catch { /* keep existing data */ }
    this.loading = false;
  }

  private renderChart() {
    const container = this.el.querySelector('.bc-echart') as HTMLElement;
    if (!container) return;
    container.style.height = this.height;
    if (!this.chart) {
      this.chart = echarts.init(container);
      this.chart.on('click', (params: any) => { this.lcChartClick.emit({ name: params.name, value: params.value, dataIndex: params.dataIndex }); });
    }
    let parsed: Array<{ name: string; value: number }> = [];
    try { parsed = JSON.parse(this.data); } catch {}
    const colorList = this.colors ? (() => { try { return JSON.parse(this.colors); } catch { return undefined; } })() : undefined;
    this.chart.setOption({
      title: { text: this.chartTitle, left: 'center', textStyle: { fontSize: 14 } },
      tooltip: this.tooltipEnabled ? { trigger: 'axis' } : undefined,
      legend: this.legend ? {} : undefined,
      animation: this.animate,
      xAxis: { type: 'category', data: parsed.map(d => d.name) },
      yAxis: { type: 'value' },
      series: [{ type: 'bar', data: parsed.map(d => d.value), itemStyle: colorList ? { color: (p: any) => colorList[p.dataIndex % colorList.length] } : { color: '#4f46e5' } }],
    });
  }

  @Method() async updateData(newData: unknown): Promise<void> { this.data = typeof newData === 'string' ? newData : JSON.stringify(newData); }
  @Method() async setData(newData: unknown): Promise<void> { this.data = typeof newData === 'string' ? newData : JSON.stringify(newData); }
  @Method() async refresh(): Promise<void> { if (this.dataSource || this.dataFetcher) await this._fetchData(); else this.renderChart(); }
  @Method() async resize(): Promise<void> { this.chart?.resize(); }
  @Method() async exportImage(format: string = 'png'): Promise<string> { return this.chart?.getDataURL({ type: format as 'png' | 'jpeg' | 'svg', pixelRatio: 2 }) || ''; }

  render() {
    return (
      <div class={{ 'bc-chart-wrap': true, 'bc-chart-loading': this.loading }}>
        {this.loading && <div class="bc-chart-loading-overlay"><span class="bc-field-loading-indicator" /></div>}
        <div class="bc-echart"></div>
      </div>
    );
  }
}
