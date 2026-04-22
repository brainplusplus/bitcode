import { Component, Prop, State, Event, EventEmitter, Element, Listen, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { i18n } from '../../../core/i18n';

@Component({
  tag: 'bc-view-form',
  styleUrl: 'bc-view-form.css',
  shadow: false,
})
export class BcViewForm {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() viewTitle: string = '';
  @Prop() recordId: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';

  @State() data: Record<string, unknown> = {};
  @State() loading: boolean = false;
  @State() dirty: boolean = false;

  @Event() lcFormSubmit!: EventEmitter<{model: string; data: Record<string, unknown>; id?: string}>;

  componentWillRender() {
    this.el.dir = i18n.dir;
  }

  async componentDidLoad() {
    if (this.recordId && this.model) {
      this.loading = true;
      try { const api = getApiClient(); this.data = await api.read(this.model, this.recordId); }
      catch { this.data = {}; }
      this.loading = false;
    }
  }

  @Listen('lcFieldChange')
  handleFieldChange(e: CustomEvent) {
    const { name, value } = e.detail;
    this.data = { ...this.data, [name]: value };
    this.dirty = true;
  }

  private async handleSave() {
    const api = getApiClient();
    try {
      if (this.recordId) { await api.update(this.model, this.recordId, this.data); }
      else { const r = await api.create(this.model, this.data); this.data = r; }
      this.dirty = false;
      this.lcFormSubmit.emit({ model: this.model, data: this.data, id: this.recordId });
    } catch (err) { console.error('Save failed:', err); }
  }

  render() {
    return (
      <div class="bc-view bc-view-form">
        {this.loading && <div class="bc-form-loading">{i18n.t('common.loading')}</div>}
        <div class="bc-form-body"><slot></slot></div>
        <div class="bc-form-footer">
          <button type="button" class="bc-btn bc-btn-primary" onClick={() => this.handleSave()} disabled={!this.dirty}>
            {this.recordId ? i18n.t('common.save') : i18n.t('common.create')}
          </button>
        </div>
      </div>
    );
  }
}
