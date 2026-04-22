import { Component, Prop, State, Event, EventEmitter, h } from '@stencil/core';

interface FilterCategory { label: string; field: string; options: Array<{value: string; count: number}>; }

@Component({ tag: 'bc-filter-panel', styleUrl: 'bc-filter-panel.css', shadow: false })
export class BcFilterPanel {
  @Prop({ mutable: true }) value: string = '{}';
  @Prop() categories: string = '[]';
  @Prop() placeholder: string = 'Search...';
  @State() activeFilters: Record<string, string> = {};
  @Event() lcFilterChange!: EventEmitter<{filters: Record<string, string>}>;
  @Event() lcSearch!: EventEmitter<{query: string}>;

  private getCategories(): FilterCategory[] { try { return JSON.parse(this.categories); } catch { return []; } }

  private toggleFilter(field: string, val: string) {
    const filters = { ...this.activeFilters };
    if (filters[field] === val) { delete filters[field]; } else { filters[field] = val; }
    this.activeFilters = filters;
    this.value = JSON.stringify(filters);
    this.lcFilterChange.emit({ filters });
  }

  render() {
    const cats = this.getCategories();
    return (
      <div class="bc-filter-panel">
        {cats.map(cat => (
          <div class="bc-fp-category">
            <div class="bc-fp-cat-label">{cat.label}</div>
            {cat.options.map(opt => (
              <div class={{'bc-fp-option': true, 'active': this.activeFilters[cat.field] === opt.value}} onClick={() => this.toggleFilter(cat.field, opt.value)}>
                <span class="bc-fp-opt-label">{opt.value}</span>
                <span class="bc-fp-opt-count">{opt.count}</span>
              </div>
            ))}
          </div>
        ))}
        {cats.length === 0 && <div class="bc-fp-empty">No filters available</div>}
        <slot></slot>
      </div>
    );
  }
}
