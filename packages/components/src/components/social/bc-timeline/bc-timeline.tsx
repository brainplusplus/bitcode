import { Component, Prop, State, Element, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { i18n } from '../../../core/i18n';

interface AuditEntry { id: string; field: string; oldValue: string; newValue: string; user: string; date: string; }

@Component({ tag: 'bc-timeline', styleUrl: 'bc-timeline.css', shadow: false })
export class BcTimeline {
  @Element() el!: HTMLElement;
  @Prop() recordId: string = '';
  @Prop() model: string = '';
  @State() entries: AuditEntry[] = [];
  @State() loading: boolean = false;

  componentWillRender() { this.el.dir = i18n.dir; }

  async componentDidLoad() {
    if (!this.model || !this.recordId) return;
    this.loading = true;
    try {
      const api = getApiClient();
      const res = await api.list(this.model + '_audit', { pageSize: 50 });
      this.entries = res.data.map(r => ({
        id: String(r['id'] || ''),
        field: String(r['field'] || ''),
        oldValue: String(r['old_value'] || ''),
        newValue: String(r['new_value'] || ''),
        user: String(r['user'] || 'System'),
        date: String(r['created_at'] || r['date'] || ''),
      }));
    } catch { this.entries = []; }
    this.loading = false;
  }

  private formatDate(d: string): string {
    try { return i18n.tf.date(d, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }); }
    catch { return d; }
  }

  render() {
    return (
      <div class="bc-timeline">
        <div class="bc-tl-header"><h4>{i18n.t('timeline.title')}</h4></div>
        {this.loading && <div class="bc-tl-loading">{i18n.t('common.loading')}</div>}
        <div class="bc-tl-entries">
          {this.entries.map(e => (
            <div class="bc-tl-entry">
              <div class="bc-tl-dot"></div>
              <div class="bc-tl-content">
                <span class="bc-tl-user">{e.user}</span> changed <span class="bc-tl-field">{e.field}</span>
                {e.oldValue && <span class="bc-tl-old"> from "{e.oldValue}"</span>}
                <span class="bc-tl-new"> to "{e.newValue}"</span>
                <div class="bc-tl-date">{this.formatDate(e.date)}</div>
              </div>
            </div>
          ))}
          {this.entries.length === 0 && !this.loading && <div class="bc-tl-empty">{i18n.t('timeline.noChanges')}</div>}
        </div>
      </div>
    );
  }
}
