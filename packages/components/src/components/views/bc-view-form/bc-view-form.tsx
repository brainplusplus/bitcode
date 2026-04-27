import { Component, Prop, State, Event, EventEmitter, Element, Listen, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { i18n } from '../../../core/i18n';

interface Permissions {
  can_select?: boolean;
  can_read?: boolean;
  can_write?: boolean;
  can_create?: boolean;
  can_delete?: boolean;
  can_print?: boolean;
  can_email?: boolean;
  can_report?: boolean;
  can_export?: boolean;
  can_import?: boolean;
  can_mask?: boolean;
  can_clone?: boolean;
}

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
  @Prop() permissions: string = '{}';
  @Prop() moduleName: string = '';

  @State() data: Record<string, unknown> = {};
  @State() loading: boolean = false;
  @State() dirty: boolean = false;
  @State() perms: Permissions = {};

  @Event() lcFormSubmit!: EventEmitter<{model: string; data: Record<string, unknown>; id?: string}>;

  private can(op: string): boolean {
    const key = `can_${op}` as keyof Permissions;
    return this.perms[key] !== false;
  }

  componentWillRender() {
    this.el.dir = i18n.dir;
  }

  componentWillLoad() {
    try { this.perms = JSON.parse(this.permissions); } catch { this.perms = {}; }
  }

  async componentDidLoad() {
    if (this.recordId && this.model) {
      this.loading = true;
      try {
        const api = getApiClient();
        const res = await api.read(this.model, this.recordId);
        this.data = (res as any).data || res;
      }
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

  private async handleDelete() {
    if (!this.recordId || !confirm(i18n.t('confirm.message'))) return;
    try {
      const api = getApiClient();
      await api.remove(this.model, this.recordId);
      window.history.back();
    } catch (err) { console.error('Delete failed:', err); }
  }

  private async handleClone() {
    if (!this.recordId) return;
    try {
      const url = `/api/${this.model}s/${this.recordId}/clone`;
      await fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json' } });
      await this.componentDidLoad();
    } catch (err) { console.error('Clone failed:', err); }
  }

  render() {
    const isNew = !this.recordId;
    const canSave = isNew ? this.can('create') : this.can('write');

    return (
      <div class="bc-view bc-view-form">
        {this.loading && <div class="bc-form-loading">{i18n.t('common.loading')}</div>}
        <div class="bc-form-body"><slot></slot></div>
        <div class="bc-form-footer">
          {canSave && (
            <button type="button" class="bc-btn bc-btn-primary" onClick={() => this.handleSave()} disabled={!this.dirty}>
              {isNew ? i18n.t('common.create') : i18n.t('common.save')}
            </button>
          )}
          {!isNew && this.can('delete') && (
            <button type="button" class="bc-btn bc-btn-danger" onClick={() => this.handleDelete()}>
              {i18n.t('common.delete') || 'Delete'}
            </button>
          )}
          {!isNew && this.can('clone') && (
            <button type="button" class="bc-btn bc-btn-secondary" onClick={() => this.handleClone()}>
              {i18n.t('datatable.clone') || 'Clone'}
            </button>
          )}
          {!isNew && this.can('print') && (
            <button type="button" class="bc-btn bc-btn-secondary" onClick={() => window.print()}>
              {i18n.t('common.print') || 'Print'}
            </button>
          )}
        </div>
      </div>
    );
  }
}
