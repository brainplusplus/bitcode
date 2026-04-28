import { Component, Prop, Event, EventEmitter, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-dialog-modal',
  styleUrl: 'bc-dialog-modal.css',
  shadow: false,
})
export class BcDialogModal {
  @Prop({ mutable: true }) open: boolean = false;
  @Prop() dialogTitle: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' | 'xl' = 'md';
  @Prop() loading: boolean = false;

  @Event() lcDialogClose!: EventEmitter<{type: string}>;

  @Method() async openDialog(): Promise<void> { this.open = true; }
  @Method() async closeDialog(): Promise<void> { this._close(); }
  @Method() async isOpen(): Promise<boolean> { return this.open; }

  private _close() { this.open = false; this.lcDialogClose.emit({type: 'bc-dialog-modal'}); }

  render() {
    if (!this.open) return null;
    return (
      <div class="bc-overlay" onClick={() => this._close()}>
        <div class={`bc-dialog bc-dialog-${this.size}`} onClick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label={this.dialogTitle}>
          <div class="bc-dialog-header">
            <h3>{this.dialogTitle}</h3>
            <button type="button" class="bc-close" onClick={() => this._close()}>&times;</button>
          </div>
          <div class="bc-dialog-body"><slot></slot></div>
        </div>
      </div>
    );
  }
}


