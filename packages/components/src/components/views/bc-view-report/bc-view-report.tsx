import { Component, Method, Prop, State, Element, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { i18n } from '../../../core/i18n';

@Component({ tag: 'bc-view-report', styleUrl: 'bc-view-report.css', shadow: false })
export class BcViewReport {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() viewTitle: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';
  @State() data: Array<Record<string, unknown>> = [];
  @State() loading: boolean = false;
  @State() visibleFields: string[] = [];

  private getFields(): string[] { try { return JSON.parse(this.fields); } catch { return []; } }

  componentWillRender() {
    this.el.dir = i18n.dir;
  }

  async componentDidLoad() {
    this.visibleFields = this.getFields();
    if (!this.model) return;
    this.loading = true;
    try {
      const api = getApiClient();
      const res = await api.list(this.model, { pageSize: 200 });
      this.data = res.data;
    } catch { this.data = []; }
    this.loading = false;
  }

  private isNumeric(field: string): boolean {
    if (this.data.length === 0) return false;
    const val = this.data[0][field];
    return typeof val === 'number';
  }

  private computeTotal(field: string): number {
    return this.data.reduce((sum, row) => sum + (Number(row[field]) || 0), 0);
  }

  private computeAvg(field: string): number {
    if (this.data.length === 0) return 0;
    return this.computeTotal(field) / this.data.length;
  }  @Method() async refresh(): Promise<void> { }

  render() {
    return (
      <div class="bc-view bc-view-report">
        <div class="bc-rpt-header">
          <h2>{this.viewTitle || i18n.t('report.title')}</h2>
          <div class="bc-rpt-meta">
            <span class="bc-rpt-count">{i18n.plural('report.rows', this.data.length)}</span>
            <button type="button" class="bc-rpt-export" onClick={() => { console.log('Export CSV'); }}>{i18n.t('report.exportCsv')}</button>
          </div>
        </div>
        {this.loading && <div class="bc-rpt-loading">{i18n.t('common.loading')}</div>}
        <div class="bc-rpt-table-wrap">
          <table class="bc-rpt-table">
            <thead><tr>{this.visibleFields.map(f => <th>{f}</th>)}</tr></thead>
            <tbody>
              {this.data.map(row => (
                <tr>{this.visibleFields.map(f => <td class={{'numeric': this.isNumeric(f)}}>{this.isNumeric(f) ? i18n.tf.number(Number(row[f] || 0)) : String(row[f] ?? '')}</td>)}</tr>
              ))}
            </tbody>
            <tfoot>
              <tr class="bc-rpt-totals">
                {this.visibleFields.map((f, i) => (
                  <td class={{'numeric': this.isNumeric(f)}}>
                    {i === 0 ? i18n.t('common.total') : (this.isNumeric(f) ? i18n.tf.number(this.computeTotal(f)) : '')}
                  </td>
                ))}
              </tr>
              <tr class="bc-rpt-avg">
                {this.visibleFields.map((f, i) => (
                  <td class={{'numeric': this.isNumeric(f)}}>
                    {i === 0 ? i18n.t('report.average') : (this.isNumeric(f) ? i18n.tf.number(this.computeAvg(f), {maximumFractionDigits: 2}) : '')}
                  </td>
                ))}
              </tr>
            </tfoot>
          </table>
        </div>
      </div>
    );
  }
}

