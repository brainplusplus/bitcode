import { Component, Prop, State, h } from '@stencil/core';

@Component({
  tag: 'bc-viewer-pdf',
  styleUrl: 'bc-viewer-pdf.css',
  shadow: true,
})
export class BcViewerPdf {
  @Prop() src: string = '';
  @Prop() height: string = '600px';
  @Prop() toolbar: boolean = true;
  @Prop() download: boolean = true;

  @State() loadError: boolean = false;

  private getEmbedUrl(): string {
    if (!this.src) return '';
    const separator = this.src.includes('#') ? '&' : '#';
    return `${this.src}${separator}toolbar=${this.toolbar ? '1' : '0'}`;
  }

  private handleDownload() {
    const a = document.createElement('a');
    a.href = this.src;
    a.download = this.src.split('/').pop() || 'document.pdf';
    a.target = '_blank';
    a.click();
  }

  render() {
    if (!this.src) {
      return (
        <div class="bc-viewer-pdf bc-viewer-empty">
          <div class="bc-viewer-empty-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
              <polyline points="14 2 14 8 20 8" />
              <line x1="16" y1="13" x2="8" y2="13" />
              <line x1="16" y1="17" x2="8" y2="17" />
            </svg>
          </div>
          <span class="bc-viewer-empty-text">No PDF source provided</span>
        </div>
      );
    }

    return (
      <div class="bc-viewer-pdf">
        {this.download && (
          <div class="bc-viewer-toolbar">
            <button class="bc-viewer-btn" onClick={() => this.handleDownload()} title="Download PDF">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <polyline points="7 10 12 15 17 10" />
                <line x1="12" y1="15" x2="12" y2="3" />
              </svg>
              <span>Download</span>
            </button>
          </div>
        )}
        {!this.loadError ? (
          <iframe
            class="bc-viewer-pdf-frame"
            src={this.getEmbedUrl()}
            style={{ height: this.height }}
            onError={() => { this.loadError = true; }}
          />
        ) : (
          <div class="bc-viewer-fallback">
            <object
              data={this.getEmbedUrl()}
              type="application/pdf"
              class="bc-viewer-pdf-object"
              style={{ height: this.height }}
            >
              <div class="bc-viewer-fallback-content">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                  <polyline points="14 2 14 8 20 8" />
                </svg>
                <p>Unable to display PDF inline.</p>
                <a href={this.src} target="_blank" rel="noopener noreferrer" class="bc-viewer-btn">
                  Open PDF in new tab
                </a>
              </div>
            </object>
          </div>
        )}
      </div>
    );
  }
}
