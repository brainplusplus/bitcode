import { Component, Prop, State, h } from '@stencil/core';

@Component({
  tag: 'bc-viewer-audio',
  styleUrl: 'bc-viewer-audio.css',
  shadow: true,
})
export class BcViewerAudio {
  @Prop() src: string = '';
  @Prop() type: string = '';
  @Prop() controls: boolean = true;
  @Prop() autoplay: boolean = false;
  @Prop() loop: boolean = false;
  @Prop() download: boolean = true;

  @State() isPlaying: boolean = false;
  @State() currentTime: number = 0;
  @State() duration: number = 0;
  @State() volume: number = 1;
  @State() isMuted: boolean = false;
  @State() loadError: boolean = false;

  private audioEl!: HTMLAudioElement;

  private detectMimeType(): string {
    if (this.type) return this.type;
    const ext = this.src.split('?')[0].split('.').pop()?.toLowerCase() || '';
    const mimeMap: Record<string, string> = {
      mp3: 'audio/mpeg',
      m4a: 'audio/mp4',
      aac: 'audio/aac',
      ogg: 'audio/ogg',
      oga: 'audio/ogg',
      webm: 'audio/webm',
      wav: 'audio/wav',
      flac: 'audio/flac',
    };
    return mimeMap[ext] || 'audio/mpeg';
  }

  private togglePlay() {
    if (!this.audioEl) return;
    if (this.audioEl.paused) {
      this.audioEl.play();
    } else {
      this.audioEl.pause();
    }
  }

  private handleTimeUpdate() {
    if (!this.audioEl) return;
    this.currentTime = this.audioEl.currentTime;
  }

  private handleLoadedMetadata() {
    if (!this.audioEl) return;
    this.duration = this.audioEl.duration;
  }

  private handleSeek(e: Event) {
    const input = e.target as HTMLInputElement;
    if (this.audioEl) {
      this.audioEl.currentTime = parseFloat(input.value);
    }
  }

  private handleVolumeChange(e: Event) {
    const input = e.target as HTMLInputElement;
    const vol = parseFloat(input.value);
    if (this.audioEl) {
      this.audioEl.volume = vol;
      this.audioEl.muted = vol === 0;
    }
    this.volume = vol;
    this.isMuted = vol === 0;
  }

  private toggleMute() {
    if (!this.audioEl) return;
    this.audioEl.muted = !this.audioEl.muted;
    this.isMuted = this.audioEl.muted;
    if (!this.isMuted && this.volume === 0) {
      this.volume = 0.5;
      this.audioEl.volume = 0.5;
    }
  }

  private handleDownload() {
    const a = document.createElement('a');
    a.href = this.src;
    a.download = this.src.split('/').pop() || 'audio';
    a.target = '_blank';
    a.click();
  }

  private formatTime(seconds: number): string {
    if (!isFinite(seconds)) return '0:00';
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return `${m}:${s.toString().padStart(2, '0')}`;
  }

  private getFileName(): string {
    if (!this.src) return '';
    const name = this.src.split('/').pop()?.split('?')[0] || '';
    return decodeURIComponent(name);
  }

  render() {
    if (!this.src) {
      return (
        <div class="bc-viewer-audio bc-viewer-empty">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M9 18V5l12-2v13" />
            <circle cx="6" cy="18" r="3" />
            <circle cx="18" cy="16" r="3" />
          </svg>
          <span>No audio source provided</span>
        </div>
      );
    }

    return (
      <div class="bc-viewer-audio">
        <audio
          ref={(el) => this.audioEl = el as HTMLAudioElement}
          autoplay={this.autoplay}
          loop={this.loop}
          onTimeUpdate={() => this.handleTimeUpdate()}
          onLoadedMetaData={() => this.handleLoadedMetadata()}
          onPlay={() => { this.isPlaying = true; }}
          onPause={() => { this.isPlaying = false; }}
          onError={() => { this.loadError = true; }}
        >
          <source src={this.src} type={this.detectMimeType()} />
        </audio>

        {this.loadError ? (
          <div class="bc-viewer-audio-error">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="10" />
              <line x1="15" y1="9" x2="9" y2="15" />
              <line x1="9" y1="9" x2="15" y2="15" />
            </svg>
            <span>Failed to load audio</span>
          </div>
        ) : (
          <div class="bc-viewer-audio-player">
            <button class="bc-audio-play" onClick={() => this.togglePlay()}>
              {this.isPlaying ? (
                <svg viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16" /><rect x="14" y="4" width="4" height="16" /></svg>
              ) : (
                <svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3" /></svg>
              )}
            </button>

            <div class="bc-audio-info">
              <span class="bc-audio-filename">{this.getFileName()}</span>
              <div class="bc-audio-progress-row">
                <span class="bc-audio-time">{this.formatTime(this.currentTime)}</span>
                <input
                  type="range"
                  class="bc-audio-seek"
                  min="0"
                  max={this.duration || 0}
                  value={this.currentTime}
                  step="0.1"
                  onInput={(e) => this.handleSeek(e)}
                />
                <span class="bc-audio-time">{this.formatTime(this.duration)}</span>
              </div>
            </div>

            <div class="bc-audio-volume-group">
              <button class="bc-audio-btn" onClick={() => this.toggleMute()}>
                {this.isMuted ? (
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5" /><line x1="23" y1="9" x2="17" y2="15" /><line x1="17" y1="9" x2="23" y2="15" /></svg>
                ) : (
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5" /><path d="M15.54 8.46a5 5 0 0 1 0 7.07" /></svg>
                )}
              </button>
              <input
                type="range"
                class="bc-audio-volume"
                min="0"
                max="1"
                value={this.isMuted ? 0 : this.volume}
                step="0.05"
                onInput={(e) => this.handleVolumeChange(e)}
              />
            </div>

            {this.download && (
              <button class="bc-audio-btn" onClick={() => this.handleDownload()} title="Download">
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
