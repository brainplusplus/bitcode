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
});
