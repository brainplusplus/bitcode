import { BcNative } from './bc-native';

describe('BcNative', () => {

  beforeEach(() => {
    delete (window as any).__TAURI__;
  });

  describe('getEnvironment', () => {
    it('returns browser when __TAURI__ is absent', () => {
      expect(BcNative.getEnvironment()).toBe('browser');
    });

    it('returns tauri-desktop when __TAURI__ is present on desktop UA', () => {
      (window as any).__TAURI__ = { core: { invoke: jest.fn() } };
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
        configurable: true,
      });
      expect(BcNative.getEnvironment()).toBe('tauri-desktop');
    });

    it('returns tauri-mobile when __TAURI__ is present on Android UA', () => {
      (window as any).__TAURI__ = { core: { invoke: jest.fn() } };
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (Linux; Android 13)',
        configurable: true,
      });
      expect(BcNative.getEnvironment()).toBe('tauri-mobile');
    });
  });

  describe('isTauri', () => {
    it('returns false in browser', () => {
      expect(BcNative.isTauri()).toBe(false);
    });

    it('returns true when __TAURI__ exists', () => {
      (window as any).__TAURI__ = { core: { invoke: jest.fn() } };
      expect(BcNative.isTauri()).toBe(true);
    });
  });

  describe('dbExecute (browser fallback)', () => {
    it('returns empty result in browser mode', async () => {
      const result = await BcNative.dbExecute('SELECT 1');
      expect(result).toEqual({ rowsAffected: 0 });
    });
  });

  describe('dbSelect (browser fallback)', () => {
    it('returns empty array in browser mode', async () => {
      const rows = await BcNative.dbSelect('SELECT * FROM test');
      expect(rows).toEqual([]);
    });
  });

  describe('syncData (browser fallback)', () => {
    it('returns no-op result in browser mode', async () => {
      const result = await BcNative.syncData();
      expect(result).toEqual({ success: true, synced: 0, errors: 0 });
    });
  });

  describe('requestNotificationPermission (browser fallback)', () => {
    it('returns false when Notification API is absent', async () => {
      const origNotification = (window as any).Notification;
      delete (window as any).Notification;
      const result = await BcNative.requestNotificationPermission();
      expect(result).toBe(false);
      (window as any).Notification = origNotification;
    });
  });

  describe('scanBarcode (browser fallback)', () => {
    it('throws in browser mode', async () => {
      await expect(BcNative.scanBarcode()).rejects.toThrow('Barcode scanning requires Tauri');
    });
  });

  describe('authenticate (browser fallback)', () => {
    it('returns false in browser mode', async () => {
      const result = await BcNative.authenticate();
      expect(result).toBe(false);
    });
  });

  describe('setDbPath', () => {
    it('accepts a new path without throwing', () => {
      expect(() => BcNative.setDbPath('sqlite:test.db')).not.toThrow();
    });
  });
});
