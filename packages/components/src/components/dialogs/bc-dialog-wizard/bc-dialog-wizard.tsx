import { Component, Prop, State, Event, EventEmitter, Element, h } from '@stencil/core';
import { i18n } from '../../../core/i18n';

@Component({ tag: 'bc-dialog-wizard', styleUrl: 'bc-dialog-wizard.css', shadow: false })
export class BcDialogWizard {
  @Element() el!: HTMLElement;
  @Prop({ mutable: true }) open: boolean = false;
  @Prop() dialogTitle: string = '';
  @Prop() steps: string = '[]';
  @State() currentStep: number = 0;
  @Event() lcDialogClose!: EventEmitter<{type: string; step: number}>;
  @Event() lcWizardComplete!: EventEmitter<{step: number}>;

  private getSteps(): string[] { try { return JSON.parse(this.steps); } catch { return []; } }

  componentWillRender() { this.el.dir = i18n.dir; }

  private close() { this.open = false; this.currentStep = 0; this.lcDialogClose.emit({ type: 'wizard', step: this.currentStep }); }
  private next() {
    const steps = this.getSteps();
    if (this.currentStep < steps.length - 1) { this.currentStep++; }
    else { this.lcWizardComplete.emit({ step: this.currentStep }); this.close(); }
  }
  private prev() { if (this.currentStep > 0) this.currentStep--; }

  render() {
    if (!this.open) return null;
    const steps = this.getSteps();
    const isLast = this.currentStep >= steps.length - 1;
    return (
      <div class="bc-overlay" onClick={() => this.close()}>
        <div class="bc-wizard" onClick={(e) => e.stopPropagation()}>
          <div class="bc-wizard-header">
            <h3>{this.dialogTitle}</h3>
            <button type="button" class="bc-close" onClick={() => this.close()}>{'\u00D7'}</button>
          </div>
          <div class="bc-wizard-steps">
            {steps.map((step, i) => (
              <div class={{'bc-wizard-step': true, 'active': i === this.currentStep, 'done': i < this.currentStep}}>
                <span class="bc-step-num">{i < this.currentStep ? '\u2713' : String(i + 1)}</span>
                <span class="bc-step-label">{step}</span>
              </div>
            ))}
          </div>
          <div class="bc-wizard-body"><slot name={'step-' + this.currentStep}></slot><slot></slot></div>
          <div class="bc-wizard-footer">
            <button type="button" class="bc-btn" onClick={() => this.prev()} disabled={this.currentStep === 0}>{'\u2190'} {i18n.t('wizard.back')}</button>
            <span class="bc-wizard-progress">{this.currentStep + 1} {i18n.t('common.of')} {steps.length}</span>
            <button type="button" class="bc-btn bc-btn-primary" onClick={() => this.next()}>{isLast ? i18n.t('wizard.finish') : i18n.t('wizard.next') + ' \u2192'}</button>
          </div>
        </div>
      </div>
    );
  }
}
