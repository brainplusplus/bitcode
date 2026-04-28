import { Component, Prop, State, Event, EventEmitter, Element, Method, h } from '@stencil/core';
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

  @Method() async openDialog(): Promise<void> { this.open = true; }
  @Method() async closeDialog(): Promise<void> { this._close(); }
  @Method() async goToStep(step: number): Promise<void> { this.currentStep = step; }
  @Method() async nextStep(): Promise<void> { this._next(); }
  @Method() async prevStep(): Promise<void> { this._prev(); }
  @Method() async getCurrentStep(): Promise<number> { return this.currentStep; }

  private _close() { this.open = false; this.currentStep = 0; this.lcDialogClose.emit({ type: 'wizard', step: this.currentStep }); }
  private _next() {
    const steps = this.getSteps();
    if (this.currentStep < steps.length - 1) { this.currentStep++; }
    else { this.lcWizardComplete.emit({ step: this.currentStep }); this._close(); }
  }
  private _prev() { if (this.currentStep > 0) this.currentStep--; }

  render() {
    if (!this.open) return null;
    const steps = this.getSteps();
    const isLast = this.currentStep >= steps.length - 1;
    return (
      <div class="bc-overlay" onClick={() => this._close()}>
        <div class="bc-wizard" onClick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
          <div class="bc-wizard-header">
            <h3>{this.dialogTitle}</h3>
            <button type="button" class="bc-close" onClick={() => this._close()}>{'\u00D7'}</button>
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
            <button type="button" class="bc-btn" onClick={() => this._prev()} disabled={this.currentStep === 0}>{'\u2190'} {i18n.t('wizard.back')}</button>
            <span class="bc-wizard-progress">{this.currentStep + 1} {i18n.t('common.of')} {steps.length}</span>
            <button type="button" class="bc-btn bc-btn-primary" onClick={() => this._next()}>{isLast ? i18n.t('wizard.finish') : i18n.t('wizard.next') + ' \u2192'}</button>
          </div>
        </div>
      </div>
    );
  }
}
