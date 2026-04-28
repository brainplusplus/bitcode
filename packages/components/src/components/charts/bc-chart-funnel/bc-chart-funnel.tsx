import { Component, Prop, Element, Watch, Method, Event, EventEmitter, h } from '@stencil/core';
import { ChartClickEvent, DataFetcher } from '../../../core/types';
import { fetchData } from '../../../core/data-fetcher';
import * as echarts from 'echarts';

@Component({ tag: 'bc-chart-funnel', styleUrl: 'bc-chart-funnel.css', shadow: false })
export class BcChartFunnel {
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
  private _refreshTimer: ReturnType<typeof setInterval> | null = null;
  dataFetcher?: DataFetcher;

  componentDidLoad() { this.renderChart(); if (this.dataSource || this.dataFetcher) this._fetchRemoteData(); if (this.refreshInterval > 0) this._refreshTimer = setInterval(() => this._fetchRemoteData(), this.refreshInterval); }
  @Watch('data') onDataChange() { this.renderChart(); }
  disconnectedCallback() { this.chart?.dispose(); if (this._refreshTimer) clearInterval(this._refreshTimer); }  private async _fetchRemoteData() { this.loading = true; try { const result = await fetchData({ fetcher: this.dataFetcher, element: this.el, dataSource: this.dataSource, fetchHeaders: this.fetchHeaders }); this.data = JSON.stringify(result.data); } catch {} this.loading = false; }

  private renderChart() {
    const container = this.el.querySelector('.bc-echart') as HTMLElement;
    if (!container) return;
    if (!this.chart) this.chart = echarts.init(container);
    let parsed: Array<{name: string; value: number}> = [];
    try { parsed = JSON.parse(this.data); } catch {}
    this.chart.setOption({
      title: { text: this.chartTitle, left: 'center', textStyle: { fontSize: 14 } },
      tooltip: { trigger: 'item' },
      series: [{ type: 'funnel', data: parsed, sort: 'descending' }],
    });
  }  @Method() async updateData(newData: unknown): Promise<void> { this.data = typeof newData === 'string' ? newData : JSON.stringify(newData); }
  @Method() async setData(newData: unknown): Promise<void> { this.data = typeof newData === 'string' ? newData : JSON.stringify(newData); }
  @Method() async refresh(): Promise<void> { if (this.dataSource || this.dataFetcher) await this._fetchRemoteData(); else this.renderChart(); }
  @Method() async resize(): Promise<void> { this.chart?.resize(); }
  @Method() async exportImage(format: string = 'png'): Promise<string> { return this.chart?.getDataURL({ type: format as 'png' | 'jpeg' | 'svg', pixelRatio: 2 }) || ''; }



  render() { return (<div class="bc-chart-wrap"><div class="bc-echart"></div></div>); }
}





