import { Component, Prop, State, Method, Event, EventEmitter, h } from '@stencil/core';
import { OfflineStore } from '../../../core/offline-store';

@Component({
  tag: 'bc-sync-status',
  styleUrl: 'bc-sync-status.css',
  shadow: false,
})
export class BcSyncStatus {
  @Prop() pollInterval: number = 30000;
  @Prop() compact: boolean = false;
  @Prop() showSyncButton: boolean = true;

  @State() isOnline: boolean = false;
  @State() pendingCount: number = 0;
  @State() errorCount: number = 0;
  @State() deadCount: number = 0;
  @State() lastSyncAt: string = '';
  @State() conflictCount: number = 0;
  @State() syncing: boolean = false;

  @Event() bcSyncTriggered: EventEmitter<void>;
  @Event() bcSyncCompleted: EventEmitter<{ synced: number; errors: number; applied: number; conflicts: number }>;

  private _pollTimer: ReturnType<typeof setInterval> | null = null;

  connectedCallback() {
    this.refreshStatus();
    if (this.pollInterval > 0) {
      this._pollTimer = setInterval(() => this.refreshStatus(), this.pollInterval);
    }
  }

  disconnectedCallback() {
    if (this._pollTimer) {
      clearInterval(this._pollTimer);
      this._pollTimer = null;
    }
  }

  @Method()
  async refreshStatus(): Promise<void> {
    try {
      const status = await OfflineStore.getSyncStatus();
      this.isOnline = status.isOnline;
      this.pendingCount = status.pendingCount;
      this.errorCount = status.errorCount;
      this.deadCount = status.deadCount;
      this.lastSyncAt = status.lastSyncAt;
      this.conflictCount = status.conflictCount;
    } catch {
      this.isOnline = false;
    }
  }

  @Method()
  async triggerSync(): Promise<void> {
    if (this.syncing || !this.isOnline) return;
    this.syncing = true;
    this.bcSyncTriggered.emit();

    try {
      const result = await OfflineStore.syncAll();
      this.bcSyncCompleted.emit({
        synced: result.pushResult.synced,
        errors: result.pushResult.errors,
        applied: result.pullResult.applied,
        conflicts: result.pullResult.conflicts,
      });
      await this.refreshStatus();
    } finally {
      this.syncing = false;
    }
  }

  private formatLastSync(): string {
    if (!this.lastSyncAt) return 'Never';
    const diff = Date.now() - new Date(this.lastSyncAt).getTime();
    if (diff < 60_000) return 'Just now';
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
    return `${Math.floor(diff / 86_400_000)}d ago`;
  }

  render() {
    const statusClass = this.isOnline ? 'bc-sync-online' : 'bc-sync-offline';
    const statusDot = this.isOnline ? '●' : '●';
    const statusLabel = this.isOnline ? 'Online' : 'Offline';

    if (this.compact) {
      return (
        <span class={`bc-sync-status bc-sync-compact ${statusClass}`}>
          <span class="bc-sync-dot">{statusDot}</span>
          {this.pendingCount > 0 && <span class="bc-sync-badge">{this.pendingCount}</span>}
        </span>
      );
    }

    return (
      <div class={`bc-sync-status ${statusClass}`}>
        <div class="bc-sync-indicator">
          <span class="bc-sync-dot">{statusDot}</span>
          <span class="bc-sync-label">{statusLabel}</span>
        </div>

        {this.pendingCount > 0 && (
          <div class="bc-sync-pending">
            <span class="bc-sync-count">{this.pendingCount}</span> pending
          </div>
        )}

        {this.errorCount > 0 && (
          <div class="bc-sync-errors">
            <span class="bc-sync-count bc-sync-error-count">{this.errorCount}</span> errors
          </div>
        )}

        {this.conflictCount > 0 && (
          <div class="bc-sync-conflicts">
            <span class="bc-sync-count bc-sync-conflict-count">{this.conflictCount}</span> conflicts
          </div>
        )}

        <div class="bc-sync-last">
          Last sync: {this.formatLastSync()}
        </div>

        {this.showSyncButton && this.isOnline && (
          <button
            class={`bc-sync-btn ${this.syncing ? 'bc-sync-btn-syncing' : ''}`}
            onClick={() => this.triggerSync()}
            disabled={this.syncing}
          >
            {this.syncing ? 'Syncing...' : 'Sync Now'}
          </button>
        )}
      </div>
    );
  }
}
