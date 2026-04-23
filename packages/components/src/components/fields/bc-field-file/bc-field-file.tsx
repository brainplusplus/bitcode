import { Component, Prop, State, Event, EventEmitter, h } from '@stencil/core';
import { FieldChangeEvent } from '../../../core/types';
import { getApiClient } from '../../../core/api-client';

interface UploadedFile {
  id: string;
  name: string;
  url: string;
  thumbnailUrl?: string;
  size: number;
  mimeType: string;
  progress: number;
  error?: string;
  status: 'pending' | 'uploading' | 'done' | 'error';
}

@Component({
  tag: 'bc-field-file',
  styleUrl: 'bc-field-file.css',
  shadow: true,
})
export class BcFieldFile {
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop() accept: string = '';
  @Prop() maxSize: string = '10MB';
  @Prop() multiple: boolean = false;
  @Prop() required: boolean = false;
  @Prop() disabled: boolean = false;
  @Prop() pathFormat: string = '';
  @Prop() nameFormat: string = '';
  @Prop() apiBase: string = '/api';
  @Prop() model: string = '';
  @Prop() recordId: string = '';
  @Prop() fieldName: string = '';

  @State() files: UploadedFile[] = [];
  @State() isDragging: boolean = false;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;

  private fileInput!: HTMLInputElement;

  private parseMaxSize(): number {
    const str = this.maxSize.toUpperCase();
    const num = parseFloat(str);
    if (str.endsWith('MB')) return num * 1024 * 1024;
    if (str.endsWith('KB')) return num * 1024;
    if (str.endsWith('GB')) return num * 1024 * 1024 * 1024;
    return num;
  }

  private formatSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }

  private handleDragOver(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    this.isDragging = true;
  }

  private handleDragLeave(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    this.isDragging = false;
  }

  private handleDrop(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    this.isDragging = false;
    const droppedFiles = e.dataTransfer?.files;
    if (droppedFiles) {
      this.processFiles(Array.from(droppedFiles));
    }
  }

  private handleInputChange(e: Event) {
    const input = e.target as HTMLInputElement;
    if (input.files) {
      this.processFiles(Array.from(input.files));
      input.value = '';
    }
  }

  private processFiles(fileList: File[]) {
    const maxSize = this.parseMaxSize();
    const filesToUpload = this.multiple ? fileList : [fileList[0]];

    for (const file of filesToUpload) {
      if (file.size > maxSize) {
        this.files = [...this.files, {
          id: '', name: file.name, url: '', size: file.size,
          mimeType: file.type, progress: 0, status: 'error',
          error: `File too large (max ${this.maxSize})`,
        }];
        continue;
      }
      this.uploadFile(file);
    }
  }

  private async uploadFile(file: File) {
    const entry: UploadedFile = {
      id: '', name: file.name, url: '', size: file.size,
      mimeType: file.type, progress: 0, status: 'uploading',
    };

    if (!this.multiple) {
      this.files = [entry];
    } else {
      this.files = [...this.files, entry];
    }
    const idx = this.files.length - 1;

    try {
      const form = new FormData();
      form.append('file', file);
      if (this.model) form.append('model', this.model);
      if (this.recordId) form.append('record_id', this.recordId);
      if (this.fieldName) form.append('field_name', this.fieldName);
      if (this.pathFormat) form.append('path_format', this.pathFormat);
      if (this.nameFormat) form.append('name_format', this.nameFormat);

      const client = getApiClient();
      const result = await client.uploadFile(form);

      const updated = [...this.files];
      updated[idx] = {
        ...updated[idx],
        id: result.id,
        url: result.url,
        thumbnailUrl: result.thumbnail_url || undefined,
        progress: 100,
        status: 'done',
      };
      this.files = updated;

      this.emitChange();
    } catch (err) {
      const updated = [...this.files];
      updated[idx] = {
        ...updated[idx],
        progress: 0,
        status: 'error',
        error: err instanceof Error ? err.message : 'Upload failed',
      };
      this.files = updated;
    }
  }

  private removeFile(index: number) {
    this.files = this.files.filter((_, i) => i !== index);
    this.emitChange();
  }

  private emitChange() {
    const ids = this.files.filter(f => f.status === 'done').map(f => f.id);
    const oldValue = this.value;
    this.value = this.multiple ? JSON.stringify(ids) : (ids[0] || '');
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue });
  }

  private isImage(mimeType: string): boolean {
    return mimeType.startsWith('image/') && !mimeType.includes('svg');
  }

  render() {
    return (
      <div class="bc-field">
        {this.label && (
          <label class="bc-field-label">
            {this.label}
            {this.required && <span class="required">*</span>}
          </label>
        )}

        <div
          class={{ 'bc-file-dropzone': true, 'dragging': this.isDragging, 'disabled': this.disabled }}
          onDragOver={(e) => this.handleDragOver(e)}
          onDragLeave={(e) => this.handleDragLeave(e)}
          onDrop={(e) => this.handleDrop(e)}
          onClick={() => !this.disabled && this.fileInput?.click()}
        >
          <input
            type="file"
            ref={(el) => this.fileInput = el as HTMLInputElement}
            accept={this.accept}
            multiple={this.multiple}
            disabled={this.disabled}
            onChange={(e) => this.handleInputChange(e)}
            style={{ display: 'none' }}
          />
          <div class="bc-file-dropzone-content">
            <svg class="bc-file-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="17 8 12 3 7 8" />
              <line x1="12" y1="3" x2="12" y2="15" />
            </svg>
            <span class="bc-file-dropzone-text">
              {this.isDragging ? 'Drop files here' : 'Click or drag files to upload'}
            </span>
            <span class="bc-file-dropzone-hint">
              {this.multiple ? 'Multiple files allowed' : 'Single file'} &middot; Max {this.maxSize}
            </span>
          </div>
        </div>

        {this.files.length > 0 && (
          <div class="bc-file-list">
            {this.files.map((file, index) => (
              <div class={{ 'bc-file-item': true, 'error': file.status === 'error' }}>
                <div class="bc-file-item-preview">
                  {file.status === 'done' && file.thumbnailUrl ? (
                    <img src={file.thumbnailUrl} alt={file.name} class="bc-file-thumb" />
                  ) : this.isImage(file.mimeType) ? (
                    <svg class="bc-file-type-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                      <circle cx="8.5" cy="8.5" r="1.5" />
                      <polyline points="21 15 16 10 5 21" />
                    </svg>
                  ) : (
                    <svg class="bc-file-type-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                      <polyline points="14 2 14 8 20 8" />
                    </svg>
                  )}
                </div>
                <div class="bc-file-item-info">
                  <span class="bc-file-item-name">{file.name}</span>
                  <span class="bc-file-item-size">{this.formatSize(file.size)}</span>
                  {file.error && <span class="bc-file-item-error">{file.error}</span>}
                </div>
                {file.status === 'uploading' && (
                  <div class="bc-file-progress">
                    <div class="bc-file-progress-bar" style={{ width: '100%' }}></div>
                  </div>
                )}
                {!this.disabled && (
                  <button class="bc-file-remove" onClick={(e) => { e.stopPropagation(); this.removeFile(index); }}>
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
                    </svg>
                  </button>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    );
  }
}
