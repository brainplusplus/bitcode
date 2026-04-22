import { Component, Prop, Event, EventEmitter, h } from '@stencil/core';

@Component({
  tag: 'bc-dialog-confirm',
  styleUrl: 'bc-dialog-confirm.css',
  shadow: true,
})
export class BcDialogConfirm {
  @Prop({ mutable: true }) open: boolean = false;
  @Prop() dialogTitle: string = '';

  @Event() lcDialogClose!: EventEmitter<{type: string}>;

  private close() { this.open = false; this.lcDialogClose.emit({type: 'bc-dialog-confirm'}); }

  render() {
    if (!this.open) return null;
    return (
      <div class="bc-overlay" onClick={() => this.close()}>
        <div class="bc-dialog" onClick={(e) => e.stopPropagation()}>
          <div class="bc-dialog-header">
            <h3>{this.dialogTitle}</h3>
            <button type="button" class="bc-close" onClick={() => this.close()}>&times;</button>
          </div>
          <div class="bc-dialog-body"><slot></slot></div>
        </div>
      </div>
    );
  }
}

