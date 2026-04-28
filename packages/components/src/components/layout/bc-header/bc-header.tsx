import { Component, Prop, Event, EventEmitter, h } from '@stencil/core';

interface HeaderButton {
  label: string;
  process?: string;
  variant?: string;
  visible?: string;
  confirm?: string;
}

@Component({
  tag: 'bc-header',
  styleUrl: 'bc-header.css',
  shadow: false,
})
export class BcHeader {
  @Prop() buttons: string = '[]';
  @Prop() statusField: string = '';
  @Prop() statusValue: string = '';
  @Prop() states: string = '[]';

  @Event() lcActionClick!: EventEmitter<{ process: string }>;

  private getButtons(): HeaderButton[] {
    try { return JSON.parse(this.buttons); } catch { return []; }
  }

  private getStates(): string[] {
    try { return JSON.parse(this.states); } catch { return []; }
  }

  private handleAction(btn: HeaderButton) {
    if (btn.process) {
      this.lcActionClick.emit({ process: btn.process });
    }
  }

  render() {
    const btns = this.getButtons();
    const stateList = this.getStates();

    return (
      <div class="bc-header">
        <div class="bc-header-actions">
          {btns.map(btn => (
            <button
              type="button"
              class={`bc-btn bc-btn-${btn.variant || 'default'}`}
              onClick={() => this.handleAction(btn)}
            >
              {btn.label}
            </button>
          ))}
        </div>
        {stateList.length > 0 && (
          <div class="bc-header-statusbar">
            {stateList.map(state => (
              <span class={{ 'bc-status-step': true, 'active': state === this.statusValue, 'done': stateList.indexOf(state) < stateList.indexOf(this.statusValue) }}>
                {state}
              </span>
            ))}
          </div>
        )}
      </div>
    );
  }
}

