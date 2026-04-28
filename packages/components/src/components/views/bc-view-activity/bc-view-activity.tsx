import { Component, Method, Prop, State, Element, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { i18n } from '../../../core/i18n';

@Component({ tag: 'bc-view-activity', styleUrl: 'bc-view-activity.css', shadow: false })
export class BcViewActivity {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() viewTitle: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';
  @State() activities: Array<Record<string, unknown>> = [];
  @State() loading: boolean = false;

  componentWillRender() { this.el.dir = i18n.dir; }

  async componentDidLoad() {
    if (!this.model) return;
    this.loading = true;
    try {
      const api = getApiClient();
      const res = await api.list(this.model, { pageSize: 50, sort: 'created_at', order: 'desc' });
      this.activities = res.data;
    } catch { this.activities = []; }
    this.loading = false;
  }

  private formatDate(d: string): string {
    try { return i18n.tf.date(d, { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' }); }
    catch { return d; }
  }  @Method() async refresh(): Promise<void> { }

  render() {
    return (
      <div class="bc-view bc-view-activity">
        <div class="bc-act-header">
          <h2>{this.viewTitle || i18n.t('activity.title')}</h2>
          <span class="bc-act-count">{i18n.plural('activity.activities', this.activities.length)}</span>
        </div>
        {this.loading && <div class="bc-act-loading">{i18n.t('common.loading')}</div>}
        <div class="bc-act-timeline">
          {this.activities.map((a, i) => (
            <div class="bc-act-item">
              <div class="bc-act-line">
                <div class={{'bc-act-dot': true, 'first': i === 0}}></div>
                {i < this.activities.length - 1 && <div class="bc-act-connector"></div>}
              </div>
              <div class="bc-act-content">
                <div class="bc-act-title">{String(a['name'] || a['title'] || a['action'] || '')}</div>
                <div class="bc-act-meta">
                  {a['user'] && <span class="bc-act-user">{String(a['user'])}</span>}
                  <span class="bc-act-date">{this.formatDate(String(a['created_at'] || a['date'] || ''))}</span>
                </div>
                {a['description'] && <div class="bc-act-desc">{String(a['description'])}</div>}
              </div>
            </div>
          ))}
          {this.activities.length === 0 && !this.loading && <div class="bc-act-empty">{i18n.t('activity.noActivities')}</div>}
        </div>
      </div>
    );
  }
}

