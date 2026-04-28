import { Component, Prop, State, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-viewer-image',
  styleUrl: 'bc-viewer-image.css',
  shadow: false,
})
export class BcViewerImage {
  @Prop() src: string = '';
  @Prop() alt: string = '';
  @Prop() width: string = '100%';
  @Prop() height: string = 'auto';
  @Prop() zoomable: boolean = true;
  @Prop() lightbox: boolean = true;
  @Prop() download: boolean = false;

  @State() showLightbox: boolean = false;
  @State() loadError: boolean = false;

  private handleClick() {
    if (this.lightbox && !this.loadError) {
      this.showLightbox = true;
    }
  }

  private closeLightbox() {
    this.showLightbox = false;
  }

  private handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      this.closeLightbox();
    }
  }

  private handleDownload() {
    const a = document.createElement('a');
    a.href = this.src;
    a.download = this.src.split('/').pop() || 'image';
    a.target = '_blank';
    a.click();
  }  @Prop() loading: boolean = false;

  @Method() async refresh(): Promise<void> { }

  render() {
    if (!this.src) {
      return (
        <div class="bc-viewer-image bc-viewer-empty">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
            <circle cx="8.5" cy="8.5" r="1.5" />
            <polyline points="21 15 16 10 5 21" />
          </svg>
          <span>No image source provided</span>
        </div>
      );
    }

    return (
      <div class="bc-viewer-image">
        <div
          class={{
            'bc-viewer-image-container': true,
            'zoomable': this.zoomable || this.lightbox,
          }}
          style={{ width: this.width, height: this.height }}
          onClick={() => this.handleClick()}
        >
          {!this.loadError ? (
            <img
              src={this.src}
              alt={this.alt}
              class="bc-viewer-image-img"
              onError={() => { this.loadError = true; }}
            />
          ) : (
            <div class="bc-viewer-image-error">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                <line x1="3" y1="3" x2="21" y2="21" />
              </svg>
              <span>Failed to load image</span>
            </div>
          )}
        </div>

        {this.download && !this.loadError && (
          <div class="bc-viewer-image-actions">
            <button class="bc-viewer-btn" onClick={() => this.handleDownload()} title="Download image">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <polyline points="7 10 12 15 17 10" />
                <line x1="12" y1="15" x2="12" y2="3" />
              </svg>
              <span>Download</span>
            </button>
          </div>
        )}

        {this.showLightbox && (
          <div
            class="bc-viewer-lightbox"
            onClick={() => this.closeLightbox()}
            onKeyDown={(e) => this.handleKeyDown(e)}
            tabindex="0"
          >
            <button class="bc-viewer-lightbox-close" onClick={() => this.closeLightbox()}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
            <img
              src={this.src}
              alt={this.alt}
              class="bc-viewer-lightbox-img"
              onClick={(e) => e.stopPropagation()}
            />
            {this.download && (
              <button
                class="bc-viewer-lightbox-download"
                onClick={(e) => { e.stopPropagation(); this.handleDownload(); }}
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                  <polyline points="7 10 12 15 17 10" />
                  <line x1="12" y1="15" x2="12" y2="3" />
                </svg>
              </button>
            )}
          </div>
        )}
      </div>
    );
  }
}


