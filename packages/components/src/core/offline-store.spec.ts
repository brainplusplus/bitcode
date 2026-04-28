import { OfflineStore } from './offline-store';
import { BcSetup } from './bc-setup';
import { BcNative } from './bc-native';

describe('OfflineStore', () => {

  beforeEach(() => {
    BcSetup.registerOfflineModels([]);
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
    it('inserts into SQLite and records outbox', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);

      const dbExecSpy = jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      const result = await OfflineStore.create('lead', { name: 'New Lead', email: 'a@b.com' });

      expect(result.success).toBe(true);
      expect(result.id).toBeTruthy();
      expect(dbExecSpy).toHaveBeenCalledTimes(2);

      const insertCall = dbExecSpy.mock.calls[0][0] as string;
      expect(insertCall).toContain('INSERT INTO crm_lead');

      const outboxCall = dbExecSpy.mock.calls[1][0] as string;
      expect(outboxCall).toContain('_off_outbox');
    });
  });

  describe('update (offline model)', () => {
    it('updates SQLite and records outbox', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);

      const dbExecSpy = jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      const result = await OfflineStore.update('lead', 'abc', { name: 'Updated' });

      expect(result.success).toBe(true);
      expect(result.id).toBe('abc');
      expect(dbExecSpy).toHaveBeenCalledTimes(2);

      const updateCall = dbExecSpy.mock.calls[0][0] as string;
      expect(updateCall).toContain('UPDATE crm_lead SET');
      expect(updateCall).toContain('WHERE id = ?');
    });
  });

  describe('delete (offline model)', () => {
    it('soft-deletes in SQLite and records outbox', async () => {
      BcSetup.registerOfflineModels(['lead']);
      OfflineStore.registerTableMap([{ name: 'lead', table_name: 'crm_lead' }]);

      const dbExecSpy = jest.spyOn(BcNative, 'dbExecute')
        .mockResolvedValue({ rowsAffected: 1 });

      const result = await OfflineStore.delete('lead', 'abc');

      expect(result.success).toBe(true);
      expect(dbExecSpy).toHaveBeenCalledTimes(2);

      const deleteCall = dbExecSpy.mock.calls[0][0] as string;
      expect(deleteCall).toContain('_off_deleted = 1');
      expect(deleteCall).not.toContain('DELETE FROM');
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
});
