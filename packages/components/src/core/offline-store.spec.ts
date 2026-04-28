import { OfflineStore } from './offline-store';
import { BcSetup } from './bc-setup';
import { BcNative } from './bc-native';

describe('OfflineStore', () => {

  beforeEach(() => {
    BcSetup.registerOfflineModels([]);
    OfflineStore.setDeviceId('');
    delete (window as any).__TAURI__;
    (global as any).fetch = jest.fn();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('find (online model)', () => {
    it('calls fetch() for online models', async () => {
      BcSetup.configure({ baseUrl: 'http://localhost:8080' });
      (global.fetch as jest.Mock).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ data: [{ id: '1', name: 'Test' }], total: 1 }),
      });

      const result = await OfflineStore.find('contact', { page: 1, pageSize: 10 });

      expect(global.fetch).toHaveBeenCalled();
      const calledUrl = (global.fetch as jest.Mock).mock.calls[0][0] as string;
      expect(calledUrl).toContain('/api/v1/contact');
      expect(result.data).toHaveLength(1);
    });
  });

  describe('find (offline model)', () => {
    it('calls BcNative.dbSelect() for offline models', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);

      const dbSelectSpy = jest.spyOn(BcNative, 'dbSelect')
        .mockResolvedValueOnce([{ cnt: 2 }])
        .mockResolvedValueOnce([{ id: '1', name: 'Lead A' }, { id: '2', name: 'Lead B' }]);

      const result = await OfflineStore.find('lead');

      expect(dbSelectSpy).toHaveBeenCalledTimes(2);
      const countCall = dbSelectSpy.mock.calls[0][0];
      expect(countCall).toContain('crm_lead');
      expect(countCall).toContain('COUNT(*)');
      expect(result.total).toBe(2);
      expect(result.data).toHaveLength(2);
    });
  });

  describe('findById (online model)', () => {
    it('calls fetch() for online models', async () => {
      BcSetup.configure({ baseUrl: 'http://localhost:8080' });
      (global.fetch as jest.Mock).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ id: '1', name: 'Contact' }),
      });

      const result = await OfflineStore.findById('contact', '1');

      expect(global.fetch).toHaveBeenCalled();
      expect(result).toEqual({ id: '1', name: 'Contact' });
    });
  });

  describe('findById (offline model)', () => {
    it('calls BcNative.dbSelect() for offline models', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);

      const dbSelectSpy = jest.spyOn(BcNative, 'dbSelect')
        .mockResolvedValueOnce([{ id: 'abc', name: 'Lead X' }]);

      const result = await OfflineStore.findById('lead', 'abc');

      expect(dbSelectSpy).toHaveBeenCalledWith(
        expect.stringContaining('crm_lead'),
        ['abc'],
      );
      expect(result).toEqual({ id: 'abc', name: 'Lead X' });
    });

    it('returns null when not found', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);

      jest.spyOn(BcNative, 'dbSelect').mockResolvedValueOnce([]);

      const result = await OfflineStore.findById('lead', 'nonexistent');
      expect(result).toBeNull();
    });
  });

  describe('create (offline model)', () => {
    it('inserts into SQLite with transaction and records outbox', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);
      OfflineStore.setDeviceId('DEV-A');

      const dbExecSpy = jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      const result = await OfflineStore.create('lead', { name: 'New Lead', email: 'a@b.com' });

      expect(result.success).toBe(true);
      expect(result.id).toBeTruthy();

      expect(dbExecSpy.mock.calls[0][0]).toBe('BEGIN TRANSACTION');

      const insertCall = dbExecSpy.mock.calls[1][0] as string;
      expect(insertCall).toContain('INSERT INTO crm_lead');

      const outboxCall = dbExecSpy.mock.calls[2][0] as string;
      expect(outboxCall).toContain('_off_outbox');
      expect(outboxCall).toContain('device_id');

      expect(dbExecSpy.mock.calls[3][0]).toBe('COMMIT');
    });

    it('rolls back on failure', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);

      const dbExecSpy = jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValueOnce({ rowsAffected: 0 })
        .mockRejectedValueOnce(new Error('UNIQUE constraint failed'))
        .mockResolvedValueOnce({ rowsAffected: 0 });

      await expect(OfflineStore.create('lead', { name: 'Dup' })).rejects.toThrow('UNIQUE constraint');
      expect(dbExecSpy.mock.calls[2][0]).toBe('ROLLBACK');
    });
  });

  describe('update (offline model)', () => {
    it('updates SQLite with transaction and increments _off_version', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);

      const dbExecSpy = jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      const result = await OfflineStore.update('lead', 'abc', { name: 'Updated' });

      expect(result.success).toBe(true);
      expect(result.id).toBe('abc');

      expect(dbExecSpy.mock.calls[0][0]).toBe('BEGIN TRANSACTION');

      const updateCall = dbExecSpy.mock.calls[1][0] as string;
      expect(updateCall).toContain('UPDATE crm_lead SET');
      expect(updateCall).toContain('_off_version = _off_version + 1');
      expect(updateCall).toContain('WHERE id = ?');

      expect(dbExecSpy.mock.calls[3][0]).toBe('COMMIT');
    });
  });

  describe('delete (offline model)', () => {
    it('soft-deletes in SQLite with transaction and records outbox', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);

      const dbExecSpy = jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      const result = await OfflineStore.delete('lead', 'abc');

      expect(result.success).toBe(true);

      expect(dbExecSpy.mock.calls[0][0]).toBe('BEGIN TRANSACTION');

      const deleteCall = dbExecSpy.mock.calls[1][0] as string;
      expect(deleteCall).toContain('_off_deleted = 1');
      expect(deleteCall).toContain('_off_version = _off_version + 1');
      expect(deleteCall).not.toContain('DELETE FROM');

      expect(dbExecSpy.mock.calls[3][0]).toBe('COMMIT');
    });
  });

  describe('SQL injection prevention', () => {
    it('rejects invalid column names in filters', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);

      jest.spyOn(BcNative, 'dbSelect').mockResolvedValue([{ cnt: 0 }]);

      await expect(
        OfflineStore.find('lead', { filters: { 'name; DROP TABLE crm_lead--': 'x' } }),
      ).rejects.toThrow('Invalid column name');
    });

    it('rejects invalid sort field names', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);

      jest.spyOn(BcNative, 'dbSelect')
        .mockResolvedValueOnce([{ cnt: 0 }]);

      await expect(
        OfflineStore.find('lead', { sort: [{ field: '1=1; --', direction: 'asc' }] }),
      ).rejects.toThrow('Invalid column name');
    });

    it('allows valid column names', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);

      const dbSelectSpy = jest.spyOn(BcNative, 'dbSelect')
        .mockResolvedValueOnce([{ cnt: 1 }])
        .mockResolvedValueOnce([{ id: '1', name: 'Test' }]);

      const result = await OfflineStore.find('lead', { filters: { name: 'Test' } });
      expect(result.data).toHaveLength(1);
    });
  });

  describe('envelope grouping', () => {
    it('groups operations under same envelope_id within a transaction', async () => {
      BcSetup.registerOfflineModels(['sale', 'sale_item']);
      OfflineStore.registerTableMap([
        { name: 'sale', table_name: 'pos_sale' },
        { name: 'sale_item', table_name: 'pos_sale_item' },
      ]);

      const dbExecSpy = jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      const envId = OfflineStore.beginTransaction();
      await OfflineStore.create('sale', { total: 100 });
      await OfflineStore.create('sale_item', { product: 'Widget', qty: 2 });
      OfflineStore.commitTransaction();

      const outboxCalls = dbExecSpy.mock.calls.filter(
        c => (c[0] as string).includes('_off_outbox'),
      );
      expect(outboxCalls).toHaveLength(2);

      const env1 = (outboxCalls[0][1] as unknown[])[0];
      const env2 = (outboxCalls[1][1] as unknown[])[0];
      expect(env1).toBe(envId);
      expect(env2).toBe(envId);
    });
  });

  describe('isModelOffline via BcSetup', () => {
    it('returns false for unregistered models', () => {
      expect(BcSetup.isModelOffline('contact')).toBe(false);
    });

    it('returns true for registered offline models', () => {
      BcSetup.registerOfflineModels(['lead', 'sale']);
      expect(BcSetup.isModelOffline('lead')).toBe(true);
      expect(BcSetup.isModelOffline('sale')).toBe(true);
      expect(BcSetup.isModelOffline('contact')).toBe(false);
    });
  });

  describe('registerTableMap', () => {
    it('maps model names to table names', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);

      const dbSelectSpy = jest.spyOn(BcNative, 'dbSelect')
        .mockResolvedValueOnce([{ cnt: 0 }])
        .mockResolvedValueOnce([]);

      await OfflineStore.find('lead');

      const sql = dbSelectSpy.mock.calls[0][0] as string;
      expect(sql).toContain('crm_lead');
      expect(sql).not.toContain(' lead ');
    });
  });

  describe('device registration', () => {
    it('sets device_id after setDeviceId()', () => {
      OfflineStore.setDeviceId('DEV-B');
      expect(OfflineStore.getDeviceId()).toBe('DEV-B');
    });
  });

  describe('create populates _off_hlc', () => {
    it('includes _off_hlc in INSERT when device_id is set', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);
      OfflineStore.setDeviceId('DEV-HLC');

      const dbExecSpy = jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      await OfflineStore.create('lead', { name: 'HLC Test' });

      const insertCall = dbExecSpy.mock.calls[1][0] as string;
      expect(insertCall).toContain('_off_hlc');

      const insertVals = dbExecSpy.mock.calls[1][1] as unknown[];
      const hlcIdx = insertCall.split('(')[1].split(')')[0].split(',').findIndex(
        (col: string) => col.trim() === '_off_hlc',
      );
      expect(hlcIdx).toBeGreaterThanOrEqual(0);
      const hlcVal = insertVals[hlcIdx] as string;
      expect(hlcVal).toContain('DEV-HLC');
      expect(hlcVal.split(':').length).toBeGreaterThanOrEqual(3);
    });
  });

  describe('syncPull with conflict detection', () => {
    it('returns conflicts count when local record has pending changes', async () => {
      BcSetup.registerOfflineModels(['product']);
      OfflineStore.registerTableMap([{ name: 'product', table_name: 'pos_product' }]);
      OfflineStore.setDeviceId('DEV-A');

      const dbSelectSpy = jest.spyOn(BcNative, 'dbSelect');
      const dbExecSpy = jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      dbSelectSpy
        .mockResolvedValueOnce([{ last_pull_version: 0 }])
        .mockResolvedValueOnce([{
          id: 'rec-1', name: 'Widget', price: 150,
          _off_status: 'PENDING', _off_version: 3, _off_hlc: 'zzzzzz:0001:DEV-A',
          _off_deleted: 0,
        }]);

      (global.fetch as jest.Mock).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({
          changes: [{
            table_name: 'pos_product',
            record_id: 'rec-1',
            operation: 'UPDATE',
            data: { id: 'rec-1', price: 120 },
            version: 5,
            hlc: 'aaaaaa:0001:DEV-B',
          }],
          max_version: 5,
        }),
      });

      BcSetup.configure({ baseUrl: 'http://localhost:8080' });
      const result = await OfflineStore.syncPull();

      expect(result.applied).toBe(1);
      expect(result.conflicts).toBe(1);

      const conflictLogCalls = dbExecSpy.mock.calls.filter(
        c => (c[0] as string).includes('_off_conflict_log'),
      );
      expect(conflictLogCalls.length).toBeGreaterThanOrEqual(1);
    });

    it('applies remote changes directly when no local conflict', async () => {
      BcSetup.registerOfflineModels(['product']);
      OfflineStore.registerTableMap([{ name: 'product', table_name: 'pos_product' }]);
      OfflineStore.setDeviceId('DEV-A');

      const dbSelectSpy = jest.spyOn(BcNative, 'dbSelect');
      jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      dbSelectSpy
        .mockResolvedValueOnce([{ last_pull_version: 0 }])
        .mockResolvedValueOnce([{
          id: 'rec-2', name: 'Gadget', price: 100,
          _off_status: 'SYNCED', _off_version: 1,
          _off_deleted: 0,
        }]);

      (global.fetch as jest.Mock).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({
          changes: [{
            table_name: 'pos_product',
            record_id: 'rec-2',
            operation: 'UPDATE',
            data: { id: 'rec-2', price: 200 },
            version: 3,
          }],
          max_version: 3,
        }),
      });

      BcSetup.configure({ baseUrl: 'http://localhost:8080' });
      const result = await OfflineStore.syncPull();

      expect(result.applied).toBe(1);
      expect(result.conflicts).toBe(0);
    });

    it('edit wins over delete when local has pending changes', async () => {
      BcSetup.registerOfflineModels(['product']);
      OfflineStore.registerTableMap([{ name: 'product', table_name: 'pos_product' }]);
      OfflineStore.setDeviceId('DEV-A');

      const dbSelectSpy = jest.spyOn(BcNative, 'dbSelect');
      const dbExecSpy = jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      dbSelectSpy
        .mockResolvedValueOnce([{ last_pull_version: 0 }])
        .mockResolvedValueOnce([{
          id: 'rec-3', name: 'Edited', price: 999,
          _off_status: 'PENDING', _off_version: 2,
          _off_deleted: 0,
        }]);

      (global.fetch as jest.Mock).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({
          changes: [{
            table_name: 'pos_product',
            record_id: 'rec-3',
            operation: 'DELETE',
            data: { id: 'rec-3' },
            version: 4,
          }],
          max_version: 4,
        }),
      });

      BcSetup.configure({ baseUrl: 'http://localhost:8080' });
      const result = await OfflineStore.syncPull();

      expect(result.applied).toBe(1);
      expect(result.conflicts).toBe(1);

      const deleteCalls = dbExecSpy.mock.calls.filter(
        c => (c[0] as string).includes('_off_deleted = 1'),
      );
      expect(deleteCalls.length).toBe(0);
    });
  });

  describe('getNextReceiptNumber', () => {
    it('generates sequential receipt numbers using device prefix', async () => {
      const dbSelectSpy = jest.spyOn(BcNative, 'dbSelect');
      const dbExecSpy = jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      dbSelectSpy
        .mockResolvedValueOnce([{ prefix: '001-A', last_sequence: 15 }]);

      const num = await OfflineStore.getNextReceiptNumber('pos_sale');

      expect(num).toBe('001-A-0016');

      const upsertCall = dbExecSpy.mock.calls[0][0] as string;
      expect(upsertCall).toContain('_off_number_sequence');
      expect(upsertCall).toContain('ON CONFLICT');
    });

    it('falls back to device_prefix from _off_sync_state when no sequence exists', async () => {
      const dbSelectSpy = jest.spyOn(BcNative, 'dbSelect');
      jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      dbSelectSpy
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([{ device_prefix: '042-B' }]);

      const num = await OfflineStore.getNextReceiptNumber('pos_sale');

      expect(num).toBe('042-B-0001');
    });

    it('uses default prefix when no state exists', async () => {
      const dbSelectSpy = jest.spyOn(BcNative, 'dbSelect');
      jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      dbSelectSpy
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([]);

      const num = await OfflineStore.getNextReceiptNumber('pos_sale');

      expect(num).toBe('000-X-0001');
    });

    it('rejects invalid table names', async () => {
      await expect(
        OfflineStore.getNextReceiptNumber('DROP TABLE; --'),
      ).rejects.toThrow('Invalid table name');
    });
  });

  describe('configureSyncOptions', () => {
    it('sets batch size within bounds', () => {
      OfflineStore.configureSyncOptions({ batchSize: 50 });
    });

    it('clamps batch size to valid range without throwing', () => {
      OfflineStore.configureSyncOptions({ batchSize: 0 });
      OfflineStore.configureSyncOptions({ batchSize: 10000 });
    });
  });

  describe('cacheAuth', () => {
    it('caches auth credentials from server', async () => {
      OfflineStore.setDeviceId('DEV-AUTH');
      BcSetup.configure({ baseUrl: 'http://localhost:8080' });

      (global.fetch as jest.Mock).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({
          user_id: 'usr-001',
          username: 'admin',
          email: 'admin@test.com',
          groups: ['admin', 'manager'],
          user_hash: 'abc123hash',
          cached_at: '2026-05-01T10:00:00Z',
          expires_at: '2026-05-04T10:00:00Z',
        }),
      });

      const dbExecSpy = jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      const result = await OfflineStore.cacheAuth(undefined, 'admin', 'password123');

      expect(result).not.toBeNull();
      expect(result!.userId).toBe('usr-001');
      expect(result!.username).toBe('admin');
      expect(result!.groups).toEqual(['admin', 'manager']);

      const authCacheCall = dbExecSpy.mock.calls.find(
        c => (c[0] as string).includes('_off_auth_cache'),
      );
      expect(authCacheCall).toBeTruthy();
    });

    it('returns null when server is unreachable', async () => {
      OfflineStore.setDeviceId('DEV-AUTH');
      BcSetup.configure({ baseUrl: 'http://localhost:8080' });

      (global.fetch as jest.Mock).mockRejectedValue(new Error('Network error'));

      const result = await OfflineStore.cacheAuth(undefined, 'admin', 'password');
      expect(result).toBeNull();
    });

    it('returns null when credentials are missing', async () => {
      const result = await OfflineStore.cacheAuth(undefined, '', '');
      expect(result).toBeNull();
    });
  });

  describe('authenticateOffline', () => {
    it('rejects when no cached credentials exist', async () => {
      OfflineStore.setDeviceId('DEV-AUTH');

      jest.spyOn(BcNative, 'dbSelect')
        .mockResolvedValueOnce([{ failed_auth_attempts: 0, locked_until: '' }])
        .mockResolvedValueOnce([]);

      const result = await OfflineStore.authenticateOffline('admin', 'password');
      expect(result.success).toBe(false);
      expect(result.error).toContain('No cached credentials');
    });

    it('rejects when offline session has expired', async () => {
      OfflineStore.setDeviceId('DEV-AUTH');

      const pastDate = new Date(Date.now() - 86_400_000).toISOString();

      jest.spyOn(BcNative, 'dbSelect')
        .mockResolvedValueOnce([{ failed_auth_attempts: 0, locked_until: '' }])
        .mockResolvedValueOnce([{
          user_id: 'usr-001',
          user_hash: 'somehash',
          user_name: 'admin',
          expires_at: pastDate,
        }]);

      const result = await OfflineStore.authenticateOffline('admin', 'password');
      expect(result.success).toBe(false);
      expect(result.error).toContain('expired');
    });

    it('rejects when account is locked', async () => {
      OfflineStore.setDeviceId('DEV-AUTH');

      const futureDate = new Date(Date.now() + 600_000).toISOString();

      jest.spyOn(BcNative, 'dbSelect')
        .mockResolvedValueOnce([{ failed_auth_attempts: 5, locked_until: futureDate }]);

      const result = await OfflineStore.authenticateOffline('admin', 'password');
      expect(result.success).toBe(false);
      expect(result.error).toContain('locked');
    });
  });

  describe('getSyncStatus', () => {
    it('returns sync status with counts', async () => {
      OfflineStore.setDeviceId('DEV-STATUS');
      BcSetup.configure({ baseUrl: 'http://localhost:8080' });

      (global.fetch as jest.Mock).mockRejectedValue(new Error('offline'));

      jest.spyOn(BcNative, 'dbSelect')
        .mockResolvedValueOnce([{ cnt: 5 }])
        .mockResolvedValueOnce([{ cnt: 2 }])
        .mockResolvedValueOnce([{ cnt: 1 }])
        .mockResolvedValueOnce([{ last_sync_at: '2026-05-01T10:00:00Z' }])
        .mockResolvedValueOnce([{ cnt: 3 }]);

      const status = await OfflineStore.getSyncStatus();

      expect(status.isOnline).toBe(false);
      expect(status.pendingCount).toBe(5);
      expect(status.errorCount).toBe(2);
      expect(status.deadCount).toBe(1);
      expect(status.lastSyncAt).toBe('2026-05-01T10:00:00Z');
      expect(status.conflictCount).toBe(3);
    });
  });

  describe('syncPush with batch limit', () => {
    it('limits outbox query to configured batch size', async () => {
      BcSetup.configure({ baseUrl: 'http://localhost:8080' });
      OfflineStore.setDeviceId('DEV-BATCH');
      OfflineStore.configureSyncOptions({ batchSize: 50 });

      const dbSelectSpy = jest.spyOn(BcNative, 'dbSelect')
        .mockResolvedValueOnce([]);

      await OfflineStore.syncPush();

      const sql = dbSelectSpy.mock.calls[0][0] as string;
      expect(sql).toContain('LIMIT');
      const params = dbSelectSpy.mock.calls[0][1] as unknown[];
      expect(params[0]).toBe(50);
    });
  });
});
