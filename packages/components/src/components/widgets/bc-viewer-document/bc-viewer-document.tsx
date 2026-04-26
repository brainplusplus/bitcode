import { Component, Prop, State, h } from '@stencil/core';

@Component({
  tag: 'bc-viewer-document',
  styleUrl: 'bc-viewer-document.css',
  shadow: true,
})
export class BcViewerDocument {
  @Prop() src: string = '';
  @Prop() height: string = '600px';
  @Prop() provider: 'microsoft' | 'google' = 'microsoft';
  @Prop() download: boolean = true;

  @State() loadError: boolean = false;

  private getEmbedUrl(): string {
    if (!this.src) return '';
    const encoded = encodeURIComponent(this.src);
    if (this.provider === 'google') {
      return `https://docs.google.com/gview?url=${encoded}&embedded=true`;
    }
    return `https://view.officeapps.live.com/op/embed.aspx?src=${encoded}`;
  }

  private getFileExtension(): string {
    const url = this.src.split('?')[0];
    const ext = url.split('.').pop()?.toLowerCase() || '';
    return ext;
  }

  private getFileTypeLabel(): string {
    const ext = this.getFileExtension();
    const labels: Record<string, string> = {
      doc: 'Word Document',
      docx: 'Word Document',
      xls: 'Excel Spreadsheet',
      xlsx: 'Excel Spreadsheet',
      ppt: 'PowerPoint Presentation',
      pptx: 'PowerPoint Presentation',
      odt: 'OpenDocument Text',
      ods: 'OpenDocument Spreadsheet',
      odp: 'OpenDocument Presentation',
    };
    return labels[ext] || 'Document';
  }

  private isPublicUrl(): boolean {
    if (!this.src) return false;
    return this.src.startsWith('http://') || this.src.startsWith('https://');
  }

  private handleDownload() {
    const a = document.createElement('a');
    a.href = this.src;
    a.download = this.src.split('/').pop() || 'document';
    a.target = '_blank';
    a.click();
  }

  render() {
    if (!this.src) {
      return (
        <div class="bc-viewer-doc bc-viewer-empty">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
            <polyline points="14 2 14 8 20 8" />
          </svg>
          <span>No document source provided</span>
        </div>
      );
    }

    if (!this.isPublicUrl()) {
      return (
        <div class="bc-viewer-doc bc-viewer-local">
          <div class="bc-viewer-local-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
              <polyline points="14 2 14 8 20 8" />
            </svg>
          </div>
          <span class="bc-viewer-local-type">{this.getFileTypeLabel()}</span>
          <span class="bc-viewer-local-name">{this.src.split('/').pop()}</span>
          <span class="bc-viewer-local-hint">Preview not available for local files</span>
          {this.download && (
            <button class="bc-viewer-btn" onClick={() => this.handleDownload()}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <polyline points="7 10 12 15 17 10" />
                <line x1="12" y1="15" x2="12" y2="3" />
              </svg>
              <span>Download</span>
            </button>
          )}
        </div>
      );
    }

    return (
      <div class="bc-viewer-doc">
        {this.download && (
          <div class="bc-viewer-toolbar">
            <span class="bc-viewer-toolbar-label">{this.getFileTypeLabel()}</span>
            <button class="bc-viewer-btn" onClick={() => this.handleDownload()}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <polyline points="7 10 12 15 17 10" />
                <line x1="12" y1="15" x2="12" y2="3" />
              </svg>
              <span>Download</span>
            </button>
          </div>
        )}
        <iframe
          class="bc-viewer-doc-frame"
          src={this.getEmbedUrl()}
          style={{ height: this.height }}
          onError={() => { this.loadError = true; }}
        />
      </div>
    );
  }
}
