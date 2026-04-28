import { Component, State, Method, h, Element } from '@stencil/core';
import { i18n } from '../../../core/i18n';

@Component({
  tag: 'bc-tabs',
  styleUrl: 'bc-tabs.css',
  shadow: false,
})
export class BcTabs {
  @Element() el!: HTMLElement;
  @State() activeIndex: number = 0;

  private getTabs(): Element[] {
    return Array.from(this.el.querySelectorAll('bc-tab'));
  }

  @Method() async selectTab(index: number): Promise<void> { this._selectTab(index); }
  @Method() async getActiveIndex(): Promise<number> { return this.activeIndex; }

  private _selectTab(index: number) {
    this.activeIndex = index;
    const tabs = this.getTabs();
    tabs.forEach((tab, i) => {
      (tab as HTMLElement).style.display = i === index ? '' : 'none';
    });
  }

  componentDidLoad() {
    this._selectTab(0);
  }

  render() {
    const tabs = this.getTabs();
    const labels = tabs.map(tab => tab.getAttribute('label') || i18n.t('tab.default'));

    return (
      <div class="bc-tabs">
        <div class="bc-tabs-nav" role="tablist">
          {labels.map((label, i) => (
            <button
              type="button"
              class={{ 'bc-tab-btn': true, 'active': i === this.activeIndex }}
              role="tab"
              aria-selected={String(i === this.activeIndex)}
              onClick={() => this._selectTab(i)}
            >
              {label}
            </button>
          ))}
        </div>
        <div class="bc-tabs-content">
          <slot></slot>
        </div>
      </div>
    );
  }
}

