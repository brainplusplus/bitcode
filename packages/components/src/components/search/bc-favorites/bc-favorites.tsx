import { Component, Prop, State, Event, EventEmitter, h } from '@stencil/core';

interface Favorite { id: string; name: string; filters: Record<string, string>; }

@Component({ tag: 'bc-favorites', styleUrl: 'bc-favorites.css', shadow: false })
export class BcFavorites {
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = 'Search...';
  @State() favorites: Favorite[] = [];
  @State() showSave: boolean = false;
  @State() saveName: string = '';
  @Event() lcFavoriteSelect!: EventEmitter<{filters: Record<string, string>}>;
  @Event() lcSearch!: EventEmitter<{query: string}>;

  private save() {
    if (!this.saveName.trim()) return;
    const fav: Favorite = { id: Date.now().toString(36), name: this.saveName, filters: {} };
    try { fav.filters = JSON.parse(this.value); } catch {}
    this.favorites = [...this.favorites, fav];
    this.saveName = ''; this.showSave = false;
  }

  private select(fav: Favorite) {
    this.value = JSON.stringify(fav.filters);
    this.lcFavoriteSelect.emit({ filters: fav.filters });
  }

  private remove(id: string) {
    this.favorites = this.favorites.filter(f => f.id !== id);
  }

  render() {
    return (
      <div class="bc-favorites">
        <div class="bc-fav-header">
          <span class="bc-fav-label">Favorites</span>
          <button type="button" class="bc-fav-save-btn" onClick={() => { this.showSave = !this.showSave; }}>{'\u2606'} Save</button>
        </div>
        {this.showSave && (
          <div class="bc-fav-save-form">
            <input type="text" class="bc-fav-input" placeholder="Favorite name..." value={this.saveName} onInput={(e: Event) => { this.saveName = (e.target as HTMLInputElement).value; }} />
            <button type="button" class="bc-fav-add" onClick={() => this.save()}>Save</button>
          </div>
        )}
        <div class="bc-fav-list">
          {this.favorites.map(fav => (
            <div class="bc-fav-item">
              <button type="button" class="bc-fav-name" onClick={() => this.select(fav)}>{'\u2605'} {fav.name}</button>
              <button type="button" class="bc-fav-remove" onClick={() => this.remove(fav.id)}>{'\u00D7'}</button>
            </div>
          ))}
        </div>
        <slot></slot>
      </div>
    );
  }
}
