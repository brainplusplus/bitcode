import { Component, Prop, State, Event, EventEmitter, Watch, h } from '@stencil/core';

@Component({ tag: 'bc-toast', styleUrl: 'bc-toast.css', shadow: false })
export class BcToast {
  @Prop({ mutable: true }) open: boolean = false;
  @Prop() dialogTitle: string = '';
  @Prop() message: string = '';
  @Prop() variant: string = 'info';
  @Prop() duration: number = 4000;
  @Prop() position: string = 'top-right';
  @State() visible: boolean = false;
  @Event() lcDialogClose!: EventEmitter<{type: string}>;
  private timer: ReturnType<typeof setTimeout> | null = null;

  @Watch('open')
  onOpenChange(val: boolean) {
    if (val) { this.visible = true; this.startTimer(); }
    else { this.visible = false; }
  }

  componentDidLoad() { if (this.open) { this.visible = true; this.startTimer(); } }

  private startTimer() {
    if (this.timer) clearTimeout(this.timer);
    if (this.duration > 0) {
      this.timer = setTimeout(() => { this.close(); }, this.duration);
    }
  }

  private close() { this.visible = false; this.open = false; this.lcDialogClose.emit({ type: 'toast' }); }

  private icon(): string {
    switch (this.variant) { case 'success': return '\u2713'; case 'error': return '\u2717'; case 'warning': return '\u26A0'; default: return '\u2139'; }
  }

  render() {
    if (!this.visible) return null;
    return (
      <div class={'bc-toast-container bc-toast-' + this.position}>
        <div class={'bc-toast bc-toast-' + this.variant}>
          <span class="bc-toast-icon">{this.icon()}</span>
          <div class="bc-toast-body">
            {this.dialogTitle && <div class="bc-toast-title">{this.dialogTitle}</div>}
            <div class="bc-toast-message">{this.message}</div>
          </div>
          <button type="button" class="bc-toast-close" onClick={() => this.close()}>{'\u00D7'}</button>
        </div>
      </div>
    );
  }
}
