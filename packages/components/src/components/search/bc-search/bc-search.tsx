import { Component, Prop, State, Event, EventEmitter, Element, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { i18n } from '../../../core/i18n';

@Component({ tag: 'bc-search', styleUrl: 'bc-search.css', shadow: false })
export class BcSearch {
  @Element() el!: HTMLElement;
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = '';
  @Prop() model: string = '';
  @State() suggestions: Array<Record<string, unknown>> = [];
  @State() showSuggestions: boolean = false;
  @State() loading: boolean = false;
  @Event() lcSearch!: EventEmitter<{query: string}>;
  private debounceTimer: ReturnType<typeof setTimeout> | null = null;

  componentWillRender() { this.el.dir = i18n.dir; }

  private async handleInput(q: string) {
    this.value = q;
    this.lcSearch.emit({ query: q });
    if (this.debounceTimer) clearTimeout(this.debounceTimer);
    if (!this.model || q.length < 2) { this.suggestions = []; this.showSuggestions = false; return; }
    this.debounceTimer = setTimeout(async () => {
      this.loading = true;
      try {
        const api = getApiClient();
        this.suggestions = await api.search(this.model, q);
        this.showSuggestions = this.suggestions.length > 0;
      } catch { this.suggestions = []; this.showSuggestions = false; }
      this.loading = false;
    }, 300);
  }

  private selectSuggestion(item: Record<string, unknown>) {
    this.value = String(item['name'] || item['id'] || '');
    this.showSuggestions = false;
    this.lcSearch.emit({ query: this.value });
  }

  render() {
    return (
      <div class="bc-search-wrapper">
        <div class="bc-search-input-wrap">
          <span class="bc-search-icon">{'\uD83D\uDD0D'}</span>
          <input type="search" class="bc-search-input" value={this.value} placeholder={this.placeholder || i18n.t('common.search')} onInput={(e: Event) => this.handleInput((e.target as HTMLInputElement).value)} onFocus={() => { if (this.suggestions.length > 0) this.showSuggestions = true; }} onBlur={() => setTimeout(() => { this.showSuggestions = false; }, 200)} />
          {this.loading && <span class="bc-search-spinner">...</span>}
        </div>
        {this.showSuggestions && (
          <div class="bc-search-dropdown">
            {this.suggestions.map(item => (
              <div class="bc-search-option" onMouseDown={() => this.selectSuggestion(item)}>
                <span class="bc-search-opt-name">{String(item['name'] || item['id'] || '')}</span>
                {item['description'] && <span class="bc-search-opt-desc">{String(item['description'])}</span>}
              </div>
            ))}
          </div>
        )}
        <slot></slot>
      </div>
    );
  }
}
