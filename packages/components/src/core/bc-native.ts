/**
 * bc-native.ts — Bridge abstraction layer for BitCode native capabilities.
 *
 * Detects Tauri shell vs browser and routes calls accordingly.
 * Stencil components call BcNative; they never import Tauri APIs directly.
 *
 * Design doc: docs/plans/2026-04-28-offline-mode-design.md §4
 */

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type BcEnvironment = 'browser' | 'tauri-desktop' | 'tauri-mobile';

export interface BcPhotoOptions {
  quality?: number; // 0-100, default 80
}

export interface BcGeoPosition {
  lat: number;
  lng: number;
  accuracy?: number;
}

export interface BcDbResult {
  rowsAffected?: number;
  lastInsertId?: number;
}

export interface BcDbRow {
  [key: string]: unknown;
}

export interface BcNotifyOptions {
  title: string;
  body?: string;
}

// ---------------------------------------------------------------------------
// Tauri detection (withGlobalTauri: true in tauri.conf.json)
// ---------------------------------------------------------------------------

declare global {
  interface Window {
    __TAURI__?: {
      core?: { invoke: (cmd: string, args?: Record<string, unknown>) => Promise<unknown> };
      // Plugin namespaces populated when withGlobalTauri is enabled
      [key: string]: unknown;
    };
    __TAURI_INTERNALS__?: { ipc: unknown };
  }
}

function isTauri(): boolean {
  return typeof window !== 'undefined' && '__TAURI__' in window && window.__TAURI__ != null;
}

function tauriInvoke<T = unknown>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  const t = window.__TAURI__;
  if (!t?.core?.invoke) {
    return Promise.reject(new Error('Tauri invoke not available'));
  }
  return t.core.invoke(cmd, args) as Promise<T>;
}

// ---------------------------------------------------------------------------
// Web fallbacks
// ---------------------------------------------------------------------------

function webCameraFallback(opts?: BcPhotoOptions): Promise<string> {
  return new Promise((resolve, reject) => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = 'image/*';
    input.capture = 'environment';
    input.onchange = () => {
      const file = input.files?.[0];
      if (!file) { reject(new Error('No file selected')); return; }
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result as string;
        const base64 = result.split(',')[1] || result;
        if (opts?.quality != null && opts.quality < 100) {
          compressImage(result, opts.quality).then(resolve).catch(() => resolve(base64));
        } else {
          resolve(base64);
        }
      };
      reader.onerror = () => reject(new Error('Failed to read file'));
      reader.readAsDataURL(file);
    };
    input.click();
  });
}

function compressImage(dataUrl: string, quality: number): Promise<string> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => {
      const canvas = document.createElement('canvas');
      canvas.width = img.width;
      canvas.height = img.height;
      const ctx = canvas.getContext('2d');
      if (!ctx) { reject(new Error('Canvas not supported')); return; }
      ctx.drawImage(img, 0, 0);
      const compressed = canvas.toDataURL('image/jpeg', quality / 100);
      resolve(compressed.split(',')[1] || compressed);
    };
    img.onerror = () => reject(new Error('Failed to load image'));
    img.src = dataUrl;
  });
}

function webGeolocationFallback(): Promise<BcGeoPosition> {
  return new Promise((resolve, reject) => {
    if (!navigator.geolocation) {
      reject(new Error('Geolocation not supported'));
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => resolve({ lat: pos.coords.latitude, lng: pos.coords.longitude, accuracy: pos.coords.accuracy }),
      (err) => reject(new Error(`Geolocation error: ${err.message}`)),
      { enableHighAccuracy: true, timeout: 10000 },
    );
  });
}

function webNotificationFallback(opts: BcNotifyOptions): Promise<boolean> {
  if (!('Notification' in window)) return Promise.resolve(false);
  if (Notification.permission === 'granted') {
    new Notification(opts.title, { body: opts.body });
    return Promise.resolve(true);
  }
  if (Notification.permission === 'denied') return Promise.resolve(false);
  return Notification.requestPermission().then((perm) => {
    if (perm === 'granted') {
      new Notification(opts.title, { body: opts.body });
      return true;
    }
    return false;
  });
}

function webFileSaveFallback(_path: string, data: Uint8Array): Promise<void> {
  const blob = new Blob([data]);
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = _path.split('/').pop() || 'download';
  a.click();
  URL.revokeObjectURL(url);
  return Promise.resolve();
}

// ---------------------------------------------------------------------------
// In-memory DB fallback for browser (IndexedDB-backed key-value)
// Provides minimal SQL-like interface for browser testing only.
// Real offline apps MUST use Tauri + SQLite.
// ---------------------------------------------------------------------------

let _browserDb: IDBDatabase | null = null;
const BROWSER_STORE = 'bc_kv';

function openBrowserDb(): Promise<IDBDatabase> {
  if (_browserDb) return Promise.resolve(_browserDb);
  if (typeof indexedDB === 'undefined') return Promise.reject(new Error('IndexedDB not available'));
  return new Promise((resolve, reject) => {
    const req = indexedDB.open('bitcode_fallback', 1);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(BROWSER_STORE)) {
        db.createObjectStore(BROWSER_STORE);
      }
    };
    req.onsuccess = () => { _browserDb = req.result; resolve(_browserDb); };
    req.onerror = () => reject(new Error('IndexedDB open failed'));
  });
}

async function webDbExecuteFallback(sql: string, _params?: unknown[]): Promise<BcDbResult> {
  try { await openBrowserDb(); } catch { /* IndexedDB unavailable — still return empty result */ }
  console.warn('[BcNative] dbExecute in browser mode — SQL ignored:', sql);
  return { rowsAffected: 0 };
}

async function webDbSelectFallback(sql: string, _params?: unknown[]): Promise<BcDbRow[]> {
  try { await openBrowserDb(); } catch { /* IndexedDB unavailable — still return empty result */ }
  console.warn('[BcNative] dbSelect in browser mode — SQL ignored:', sql);
  return [];
}

// ---------------------------------------------------------------------------
// Tauri SQL helpers (uses the global plugin API)
// ---------------------------------------------------------------------------

let _tauriDbPath = 'sqlite:bitcode.db';

async function tauriDbLoad(): Promise<void> {
  await tauriInvoke('plugin:sql|load', { db: _tauriDbPath });
}

let _dbLoaded = false;

async function ensureTauriDb(): Promise<void> {
  if (_dbLoaded) return;
  await tauriDbLoad();
  _dbLoaded = true;
}

async function tauriDbExecute(sql: string, params?: unknown[]): Promise<BcDbResult> {
  await ensureTauriDb();
  const result = await tauriInvoke<{ rowsAffected: number; lastInsertId: number }>('plugin:sql|execute', {
    db: _tauriDbPath,
    query: sql,
    values: params || [],
  });
  return { rowsAffected: result.rowsAffected, lastInsertId: result.lastInsertId };
}

async function tauriDbSelect(sql: string, params?: unknown[]): Promise<BcDbRow[]> {
  await ensureTauriDb();
  return tauriInvoke<BcDbRow[]>('plugin:sql|select', {
    db: _tauriDbPath,
    query: sql,
    values: params || [],
  });
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export const BcNative = {

  getEnvironment(): BcEnvironment {
    if (!isTauri()) return 'browser';
    const ua = navigator.userAgent.toLowerCase();
    if (ua.includes('android') || ua.includes('iphone') || ua.includes('ipad')) {
      return 'tauri-mobile';
    }
    return 'tauri-desktop';
  },

  isTauri,

  async takePhoto(options?: BcPhotoOptions): Promise<string> {
    if (isTauri()) {
      return tauriInvoke<string>('plugin:camera|take_photo', { quality: options?.quality ?? 80 });
    }
    return webCameraFallback(options);
  },

  async getLocation(): Promise<BcGeoPosition> {
    if (isTauri()) {
      return tauriInvoke<BcGeoPosition>('plugin:geolocation|get_position');
    }
    return webGeolocationFallback();
  },

  async dbExecute(sql: string, params?: unknown[]): Promise<BcDbResult> {
    if (isTauri()) return tauriDbExecute(sql, params);
    return webDbExecuteFallback(sql, params);
  },

  async dbSelect(sql: string, params?: unknown[]): Promise<BcDbRow[]> {
    if (isTauri()) return tauriDbSelect(sql, params);
    return webDbSelectFallback(sql, params);
  },

  setDbPath(path: string): void {
    _tauriDbPath = path;
    _dbLoaded = false;
  },

  async scanBarcode(): Promise<string> {
    if (isTauri()) {
      return tauriInvoke<string>('plugin:barcode-scanner|scan');
    }
    throw new Error('Barcode scanning requires Tauri native shell. Use bc-field-barcode for display-only.');
  },

  async authenticate(): Promise<boolean> {
    if (isTauri()) {
      return tauriInvoke<boolean>('plugin:biometric|authenticate');
    }
    if (window.PublicKeyCredential) {
      console.warn('[BcNative] Biometric fallback to WebAuthn not yet implemented');
    }
    return Promise.resolve(false);
  },

  async saveFile(path: string, data: Uint8Array): Promise<void> {
    if (isTauri()) {
      return tauriInvoke<void>('plugin:fs|write_file', { path, contents: Array.from(data) });
    }
    return webFileSaveFallback(path, data);
  },

  async notify(options: BcNotifyOptions): Promise<boolean> {
    if (isTauri()) {
      await tauriInvoke('plugin:notification|notify', { title: options.title, body: options.body });
      return true;
    }
    return webNotificationFallback(options);
  },

  async requestNotificationPermission(): Promise<boolean> {
    if (isTauri()) {
      const granted = await tauriInvoke<boolean>('plugin:notification|is_permission_granted');
      if (granted) return true;
      const result = await tauriInvoke<string>('plugin:notification|request_permission');
      return result === 'granted';
    }
    if (!('Notification' in window)) return false;
    if (Notification.permission === 'granted') return true;
    const perm = await Notification.requestPermission();
    return perm === 'granted';
  },

  async syncData(): Promise<{ success: boolean; synced: number; errors: number }> {
    if (isTauri()) {
      return tauriInvoke<{ success: boolean; synced: number; errors: number }>('sync_data');
    }
    console.warn('[BcNative] syncData in browser mode — no-op');
    return { success: true, synced: 0, errors: 0 };
  },

  isOnline(): boolean {
    if (typeof navigator !== 'undefined' && 'onLine' in navigator) {
      return navigator.onLine;
    }
    return true;
  },

  onConnectivityChange(callback: (online: boolean) => void): () => void {
    if (typeof window === 'undefined') return () => {};
    const onOnline = () => callback(true);
    const onOffline = () => callback(false);
    window.addEventListener('online', onOnline);
    window.addEventListener('offline', onOffline);
    return () => {
      window.removeEventListener('online', onOnline);
      window.removeEventListener('offline', onOffline);
    };
  },

  getPlatformInfo(): { platform: string; isMobile: boolean; hasCamera: boolean; hasGeo: boolean } {
    const env = BcNative.getEnvironment();
    const ua = typeof navigator !== 'undefined' ? navigator.userAgent.toLowerCase() : '';
    const isMobile = env === 'tauri-mobile' || /android|iphone|ipad/.test(ua);
    const hasCamera = isTauri() || (typeof navigator !== 'undefined' && 'mediaDevices' in navigator);
    const hasGeo = typeof navigator !== 'undefined' && 'geolocation' in navigator;

    let platform = 'unknown';
    if (ua.includes('android')) platform = 'android';
    else if (ua.includes('iphone') || ua.includes('ipad')) platform = 'ios';
    else if (ua.includes('win')) platform = 'windows';
    else if (ua.includes('mac')) platform = 'macos';
    else if (ua.includes('linux')) platform = 'linux';
    else if (env === 'browser') platform = 'web';

    return { platform, isMobile, hasCamera, hasGeo };
  },
};
