import { Component, Prop, State, Event, EventEmitter, h } from '@stencil/core';

interface FilterPreset { label: string; field: string; value: string; icon?: string; }

@Component({ tag: 'bc-filter-bar', styleUrl: 'bc-filter-bar.css', shadow: false })
export class BcFilterBar {
  @Prop({ mutable: true }) value: string = '';
  @Prop() presets: string = '[]';
  @Prop() placeholder: string = 'Search...';
  @State() activePreset: string = '';
  @Event() lcFilterChange!: EventEmitter<{field: string; value: string}>;
  @Event() lcSearch!: EventEmitter<{query: string}>;

  private getPresets(): FilterPreset[] { try { return JSON.parse(this.presets); } catch { return []; } }

  private togglePreset(preset: FilterPreset) {
    if (this.activePreset === preset.label) {
      this.activePreset = '';
      this.lcFilterChange.emit({ field: '', value: '' });
    } else {
      this.activePreset = preset.label;
      this.lcFilterChange.emit({ field: preset.field, value: preset.value });
    }
  }

  render() {
    const presets = this.getPresets();
    return (
      <div class="bc-filter-bar">
        {presets.map(p => (
          <button type="button" class={{'bc-fb-btn': true, 'active': this.activePreset === p.label}} onClick={() => this.togglePreset(p)}>
            {p.icon && <span class="bc-fb-icon">{p.icon}</span>}
            {p.label}
          </button>
        ))}
        <slot></slot>
      </div>
    );
  }
}
