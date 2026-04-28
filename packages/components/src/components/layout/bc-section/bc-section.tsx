import { Component, Prop, State, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-section',
  styleUrl: 'bc-section.css',
  shadow: false,
})
export class BcSection {
  @Prop() sectionTitle: string = '';
  @Prop() description: string = '';
  @Prop() collapsible: boolean = false;
  @Prop() collapsed: boolean = false;

  @State() isCollapsed: boolean = false;

  componentWillLoad() {
    this.isCollapsed = this.collapsed;
  }

  @Method() async toggle(): Promise<void> {
    if (this.collapsible) {
      this.isCollapsed = !this.isCollapsed;
    }
  }

  @Method() async expand(): Promise<void> { this.isCollapsed = false; }
  @Method() async collapse(): Promise<void> { if (this.collapsible) this.isCollapsed = true; }

  render() {
    return (
      <section class={{ 'bc-section': true, 'is-collapsed': this.isCollapsed, 'is-collapsible': this.collapsible }}>
        {(this.sectionTitle || this.description) && (
          <div class="bc-section-header" role={this.collapsible ? 'button' : undefined} tabindex={this.collapsible ? 0 : undefined} onClick={() => this.toggle()} onKeyDown={(e) => { if (this.collapsible && (e.key === 'Enter' || e.key === ' ')) { e.preventDefault(); this.toggle(); } }}>
            <div class="bc-section-title-group">
              {this.sectionTitle && <h3 class="bc-section-title">{this.sectionTitle}</h3>}
              {this.description && <p class="bc-section-desc">{this.description}</p>}
            </div>
            {this.collapsible && (
              <span class="bc-section-toggle">{this.isCollapsed ? '\u25B6' : '\u25BC'}</span>
            )}
          </div>
        )}
        <div class="bc-section-body" style={{ display: this.isCollapsed ? 'none' : '' }}>
          <slot></slot>
        </div>
      </section>
    );
  }
}

