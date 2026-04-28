import { Component, Prop, h } from '@stencil/core';

interface SmartButton {
  label: string;
  icon?: string;
  count?: number;
  view?: string;
}

@Component({
  tag: 'bc-button-box',
  styleUrl: 'bc-button-box.css',
  shadow: false,
})
export class BcButtonBox {
  @Prop() buttons: string = '[]';

  private getButtons(): SmartButton[] {
    try { return JSON.parse(this.buttons); } catch { return []; }
  }

  render() {
    const btns = this.getButtons();
    if (btns.length === 0) return null;

    return (
      <div class="bc-button-box">
        {btns.map(btn => (
          <button type="button" class="bc-smart-btn">
            {btn.count !== undefined && <span class="bc-smart-count">{btn.count}</span>}
            <span class="bc-smart-label">{btn.label}</span>
          </button>
        ))}
      </div>
    );
  }
}

