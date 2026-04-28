import { Component, Prop, State, Event, EventEmitter, Element, Method, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { i18n } from '../../../core/i18n';

@Component({ tag: 'bc-dialog-quickentry', styleUrl: 'bc-dialog-quickentry.css', shadow: false })
export class BcDialogQuickentry {
  @Element() el!: HTMLElement;
  @Prop({ mutable: true }) open: boolean = false;
  @Prop() dialogTitle: string = '';
  @Prop() model: string = '';
  @Prop() fields: string = '[]';
  @State() formData: Record<string, string> = {};
  @State() saving: boolean = false;
  @Event() lcDialogClose!: EventEmitter<{type: string; data?: Record<string, string>}>;

  private getFields(): string[] { try { return JSON.parse(this.fields); } catch { return []; } }

  componentWillRender() { this.el.dir = i18n.dir; }

  @Method() async openDialog(): Promise<void> { this.open = true; }
  @Method() async closeDialog(): Promise<void> { this._close(); }

  private _close() { this.open = false; this.formData = {}; this.lcDialogClose.emit({ type: 'quickentry' }); }

  private async save() {
    if (!this.model) return;
    this.saving = true;
    try {
      const api = getApiClient();
      await api.create(this.model, this.formData);
      this.lcDialogClose.emit({ type: 'quickentry', data: this.formData });
      this.formData = {};
      this.open = false;
    } catch (e) { console.error('Quick create failed:', e); }
    this.saving = false;
  }

  render() {
    if (!this.open) return null;
    const fields = this.getFields();
    return (
      <div class="bc-overlay" onClick={() => this._close()}>
        <div class="bc-quickentry" onClick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
          <div class="bc-qe-header">
            <h3>{this.dialogTitle || i18n.t('quickentry.title')}</h3>
            <button type="button" class="bc-close" onClick={() => this._close()}>{'\u00D7'}</button>
          </div>
          <div class="bc-qe-body">
            {fields.map(f => (
              <div class="bc-qe-field">
                <label class="bc-qe-label">{f}</label>
                <input type="text" class="bc-qe-input" value={this.formData[f] || ''} onInput={(e: Event) => { this.formData = { ...this.formData, [f]: (e.target as HTMLInputElement).value }; }} />
              </div>
            ))}
          </div>
          <div class="bc-qe-footer">
            <button type="button" class="bc-btn" onClick={() => this._close()}>{i18n.t('common.cancel')}</button>
            <button type="button" class="bc-btn bc-btn-primary" onClick={() => this.save()} disabled={this.saving}>{this.saving ? i18n.t('quickentry.saving') : i18n.t('common.create')}</button>
          </div>
        </div>
      </div>
    );
  }
}
