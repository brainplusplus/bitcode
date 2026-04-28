import { FetchParams, FetchResult } from './types';
import { BcSetup } from './bc-setup';
import { BcNative, BcDbRow } from './bc-native';
import { buildHeaders, normalizeResponse } from './data-fetcher';

interface SaveResult {
  id: string;
  success: boolean;
}

interface DeleteResult {
  success: boolean;
}

interface SchemaFieldDef {
  name: string;
  type: string;
}

const _schemaFields = new Map<string, Set<string>>();

const OFFLINE_SYSTEM_COLUMNS = new Set([
  'id', '_off_uuid', '_off_device_id', '_off_status', '_off_version',
  '_off_deleted', '_off_created_at', '_off_updated_at', '_off_hlc', '_off_envelope_id',
]);

function registerSchemaFields(tableName: string, fields: SchemaFieldDef[]): void {
  const names = new Set<string>(OFFLINE_SYSTEM_COLUMNS);
  for (const f of fields) {
    names.add(f.name);
  }
  _schemaFields.set(tableName, names);
}

const SAFE_IDENTIFIER_RE = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

function assertSafeIdentifier(name: string, context: string): void {
  if (!SAFE_IDENTIFIER_RE.test(name)) {
    throw new Error(`[OfflineStore] Invalid ${context}: "${name}" — only alphanumeric and underscore allowed`);
  }
}

function assertValidColumn(table: string, column: string): void {
  assertSafeIdentifier(column, 'column name');
  const known = _schemaFields.get(table);
  if (known && !known.has(column)) {
    throw new Error(`[OfflineStore] Unknown column "${column}" for table "${table}"`);
  }
}

function assertSafeTable(table: string): void {
  assertSafeIdentifier(table, 'table name');
}

let _deviceId = '';
let _currentEnvelopeId: string | null = null;

function buildSelectSQL(table: string, params?: FetchParams): { sql: string; values: unknown[] } {
  assertSafeTable(table);
  const values: unknown[] = [];
  let sql = `SELECT * FROM ${table}`;
  const clauses: string[] = ['_off_deleted = 0'];

  if (params?.filters) {
    for (const [key, val] of Object.entries(params.filters)) {
      if (val === null || val === undefined) continue;
      assertValidColumn(table, key);
      clauses.push(`${key} = ?`);
      values.push(val);
    }
  }

  if (params?.search) {
    clauses.push(`(id LIKE ? OR _off_uuid LIKE ?)`);
    values.push(`%${params.search}%`, `%${params.search}%`);
  }

  if (clauses.length > 0) {
    sql += ` WHERE ${clauses.join(' AND ')}`;
  }

  if (params?.sort && params.sort.length > 0) {
    const orderParts = params.sort.map(s => {
      assertValidColumn(table, s.field);
      const dir = s.direction.toUpperCase() === 'DESC' ? 'DESC' : 'ASC';
      return `${s.field} ${dir}`;
    });
    sql += ` ORDER BY ${orderParts.join(', ')}`;
  }

  const page = params?.page ?? 1;
  const pageSize = params?.pageSize ?? 20;
  const offset = (page - 1) * pageSize;
  sql += ` LIMIT ? OFFSET ?`;
  values.push(pageSize, offset);

  return { sql, values };
}

function buildCountSQL(table: string, params?: FetchParams): { sql: string; values: unknown[] } {
  assertSafeTable(table);
  const values: unknown[] = [];
  let sql = `SELECT COUNT(*) as cnt FROM ${table}`;
  const clauses: string[] = ['_off_deleted = 0'];

  if (params?.filters) {
    for (const [key, val] of Object.entries(params.filters)) {
      if (val === null || val === undefined) continue;
      assertValidColumn(table, key);
      clauses.push(`${key} = ?`);
      values.push(val);
    }
  }

  if (params?.search) {
    clauses.push(`(id LIKE ? OR _off_uuid LIKE ?)`);
    values.push(`%${params.search}%`, `%${params.search}%`);
  }

  if (clauses.length > 0) {
    sql += ` WHERE ${clauses.join(' AND ')}`;
  }

  return { sql, values };
}

function generateUUIDv7(): string {
  const now = Date.now();
  const timeHex = now.toString(16).padStart(12, '0');

  const randomBytes = new Uint8Array(10);
  if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
    crypto.getRandomValues(randomBytes);
  } else {
    for (let i = 0; i < 10; i++) randomBytes[i] = Math.floor(Math.random() * 256);
  }

  const hex = Array.from(randomBytes, b => b.toString(16).padStart(2, '0')).join('');

  return [
    timeHex.slice(0, 8),
    timeHex.slice(8, 12),
    '7' + hex.slice(0, 3),
    ((parseInt(hex.slice(3, 5), 16) & 0x3f) | 0x80).toString(16).padStart(2, '0') + hex.slice(5, 7),
    hex.slice(7, 19),
  ].join('-');
}

function buildInsertSQL(table: string, data: Record<string, unknown>): { sql: string; values: unknown[] } {
  assertSafeTable(table);
  const keys = Object.keys(data);
  for (const k of keys) assertValidColumn(table, k);
  const placeholders = keys.map(() => '?');
  const values = keys.map(k => data[k]);
  return {
    sql: `INSERT INTO ${table} (${keys.join(', ')}) VALUES (${placeholders.join(', ')})`,
    values,
  };
}

function buildUpdateSQL(table: string, id: string, data: Record<string, unknown>): { sql: string; values: unknown[] } {
  assertSafeTable(table);
  const entries = Object.entries(data).filter(([k]) => k !== 'id');
  for (const [k] of entries) assertValidColumn(table, k);
  const setParts = entries.map(([k]) => `${k} = ?`);
  const values = entries.map(([, v]) => v);
  values.push(id);
  return {
    sql: `UPDATE ${table} SET ${setParts.join(', ')} WHERE id = ?`,
    values,
  };
}

async function recordOutbox(
  table: string,
  recordId: string,
  operation: 'CREATE' | 'UPDATE' | 'DELETE',
  payload: Record<string, unknown>,
  envelopeId?: string,
): Promise<void> {
  const idempotencyKey = `${table}:${recordId}:${operation}:${Date.now()}`;
  const envId = envelopeId || _currentEnvelopeId || generateUUIDv7();
  await BcNative.dbExecute(
    `INSERT INTO _off_outbox (envelope_id, table_name, record_id, operation, payload, idempotency_key, device_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
    [envId, table, recordId, operation, JSON.stringify(payload), idempotencyKey, _deviceId, new Date().toISOString()],
  );
}

const _tableMap = new Map<string, string>();

function resolveTable(model: string): string {
  return _tableMap.get(model) || model;
}

function detectPlatform(): string {
  if (typeof navigator === 'undefined') return 'unknown';
  const ua = navigator.userAgent.toLowerCase();
  if (ua.includes('android')) return 'android';
  if (ua.includes('iphone') || ua.includes('ipad')) return 'ios';
  if (ua.includes('win')) return 'windows';
  if (ua.includes('mac')) return 'macos';
  if (ua.includes('linux')) return 'linux';
  return 'web';
}

export const OfflineStore = {

  registerTableMap(models: Array<{ name: string; table_name: string; fields?: SchemaFieldDef[] }>): void {
    _tableMap.clear();
    for (const m of models) {
      _tableMap.set(m.name, m.table_name);
      if (m.fields) {
        registerSchemaFields(m.table_name, m.fields);
      }
    }
  },

  setDeviceId(deviceId: string): void {
    _deviceId = deviceId;
  },

  getDeviceId(): string {
    return _deviceId;
  },

  beginTransaction(): string {
    const envelopeId = generateUUIDv7();
    _currentEnvelopeId = envelopeId;
    return envelopeId;
  },

  commitTransaction(): void {
    _currentEnvelopeId = null;
  },

  async find(model: string, params?: FetchParams): Promise<FetchResult> {
    if (!BcSetup.isModelOffline(model)) {
      return fetchFromServer(model, params);
    }

    const table = resolveTable(model);
    const { sql: countSql, values: countVals } = buildCountSQL(table, params);
    const countRows = await BcNative.dbSelect(countSql, countVals);
    const total = (countRows[0] as BcDbRow)?.cnt as number ?? 0;

    const { sql, values } = buildSelectSQL(table, params);
    const rows = await BcNative.dbSelect(sql, values);

    return { data: rows as unknown[], total };
  },

  async findById(model: string, id: string): Promise<Record<string, unknown> | null> {
    if (!BcSetup.isModelOffline(model)) {
      return fetchOneFromServer(model, id);
    }

    const table = resolveTable(model);
    assertSafeTable(table);
    const rows = await BcNative.dbSelect(
      `SELECT * FROM ${table} WHERE id = ? AND _off_deleted = 0`,
      [id],
    );
    return (rows[0] as Record<string, unknown>) ?? null;
  },

  async create(model: string, data: Record<string, unknown>): Promise<SaveResult> {
    if (!BcSetup.isModelOffline(model)) {
      return saveToServer(model, data, 'POST');
    }

    const table = resolveTable(model);
    assertSafeTable(table);
    const id = (data.id as string) || generateUUIDv7();
    const now = new Date().toISOString();

    const record: Record<string, unknown> = {
      ...data,
      id,
      _off_device_id: _deviceId,
      _off_status: 'PENDING',
      _off_version: 1,
      _off_deleted: 0,
      _off_created_at: now,
      _off_updated_at: now,
    };

    await BcNative.dbExecute('BEGIN TRANSACTION', []);
    try {
      const { sql, values } = buildInsertSQL(table, record);
      await BcNative.dbExecute(sql, values);
      await recordOutbox(table, id, 'CREATE', record);
      await BcNative.dbExecute('COMMIT', []);
    } catch (err) {
      await BcNative.dbExecute('ROLLBACK', []);
      throw err;
    }

    return { id, success: true };
  },

  async update(model: string, id: string, data: Record<string, unknown>): Promise<SaveResult> {
    if (!BcSetup.isModelOffline(model)) {
      return saveToServer(model, data, 'PUT', id);
    }

    const table = resolveTable(model);
    assertSafeTable(table);
    const now = new Date().toISOString();

    const record: Record<string, unknown> = {
      ...data,
      _off_status: 'PENDING',
      _off_version: { __raw: '_off_version + 1' },
      _off_updated_at: now,
    };

    await BcNative.dbExecute('BEGIN TRANSACTION', []);
    try {
      const entries = Object.entries(record).filter(([k]) => k !== 'id');
      const setParts: string[] = [];
      const values: unknown[] = [];
      for (const [k, v] of entries) {
        assertValidColumn(table, k);
        if (v && typeof v === 'object' && (v as Record<string, unknown>).__raw) {
          setParts.push(`${k} = ${(v as Record<string, string>).__raw}`);
        } else {
          setParts.push(`${k} = ?`);
          values.push(v);
        }
      }
      values.push(id);
      const sql = `UPDATE ${table} SET ${setParts.join(', ')} WHERE id = ?`;
      await BcNative.dbExecute(sql, values);
      await recordOutbox(table, id, 'UPDATE', data);
      await BcNative.dbExecute('COMMIT', []);
    } catch (err) {
      await BcNative.dbExecute('ROLLBACK', []);
      throw err;
    }

    return { id, success: true };
  },

  async delete(model: string, id: string): Promise<DeleteResult> {
    if (!BcSetup.isModelOffline(model)) {
      return deleteFromServer(model, id);
    }

    const table = resolveTable(model);
    assertSafeTable(table);
    const now = new Date().toISOString();

    await BcNative.dbExecute('BEGIN TRANSACTION', []);
    try {
      await BcNative.dbExecute(
        `UPDATE ${table} SET _off_deleted = 1, _off_status = 'PENDING', _off_version = _off_version + 1, _off_updated_at = ? WHERE id = ?`,
        [now, id],
      );
      await recordOutbox(table, id, 'DELETE', { id });
      await BcNative.dbExecute('COMMIT', []);
    } catch (err) {
      await BcNative.dbExecute('ROLLBACK', []);
      throw err;
    }

    return { success: true };
  },

  async initFromServer(baseUrl?: string): Promise<void> {
    const url = baseUrl || BcSetup.getConfig().baseUrl;
    const headers = buildHeaders();

    try {
      const resp = await fetch(`${url}/api/v1/sync/schema`, { headers });
      if (!resp.ok) return;

      const body = await resp.json();
      const models = body.models as Array<{ name: string; table_name: string; fields?: SchemaFieldDef[] }> | undefined;
      if (!models || models.length === 0) return;

      BcSetup.registerOfflineModels(models.map(m => m.name));
      OfflineStore.registerTableMap(models);
    } catch {
      // Server unreachable — keep existing offline model config
    }
  },

  async registerDevice(baseUrl?: string, platform?: string, appVersion?: string, storeId?: string): Promise<{ deviceId: string; devicePrefix: string } | null> {
    const url = baseUrl || BcSetup.getConfig().baseUrl;
    const headers = { ...buildHeaders(), 'Content-Type': 'application/json' };

    const existing = await BcNative.dbSelect('SELECT device_id, device_prefix FROM _off_sync_state LIMIT 1', []);
    if (existing.length > 0) {
      const row = existing[0] as Record<string, string>;
      if (row.device_id) {
        _deviceId = row.device_id;
        return { deviceId: row.device_id, devicePrefix: row.device_prefix || '' };
      }
    }

    try {
      const resp = await fetch(`${url}/api/v1/sync/register`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          platform: platform || detectPlatform(),
          app_version: appVersion || '1.0.0',
          store_id: storeId || undefined,
        }),
      });
      if (!resp.ok) return null;

      const result = await resp.json() as { device_id: string; device_prefix: string; registered_at: string };
      _deviceId = result.device_id;

      await BcNative.dbExecute(
        `INSERT OR REPLACE INTO _off_sync_state (device_id, device_prefix, registered_at, last_pull_version) VALUES (?, ?, ?, 0)`,
        [result.device_id, result.device_prefix, result.registered_at],
      );

      return { deviceId: result.device_id, devicePrefix: result.device_prefix };
    } catch {
      return null;
    }
  },

  async syncPush(baseUrl?: string): Promise<{ synced: number; errors: number }> {
    const url = baseUrl || BcSetup.getConfig().baseUrl;
    const headers = { ...buildHeaders(), 'Content-Type': 'application/json' };
    let synced = 0;
    let errors = 0;

    const pending = await BcNative.dbSelect(
      `SELECT id, envelope_id, table_name, record_id, operation, payload, idempotency_key, device_id, retry_count
       FROM _off_outbox WHERE status = 'PENDING' ORDER BY id ASC`,
      [],
    ) as Array<Record<string, unknown>>;

    if (pending.length === 0) return { synced: 0, errors: 0 };

    const envelopes = new Map<string, Array<Record<string, unknown>>>();
    for (const row of pending) {
      const envId = (row.envelope_id as string) || (row.id as string).toString();
      if (!envelopes.has(envId)) envelopes.set(envId, []);
      envelopes.get(envId)!.push(row);
    }

    for (const [envelopeId, operations] of envelopes) {
      try {
        const resp = await fetch(`${url}/api/v1/sync/push`, {
          method: 'POST',
          headers,
          body: JSON.stringify({
            envelope_id: envelopeId,
            device_id: _deviceId,
            operations: operations.map(op => ({
              table_name: op.table_name,
              record_id: op.record_id,
              operation: op.operation,
              payload: JSON.parse(op.payload as string),
              idempotency_key: op.idempotency_key,
            })),
          }),
        });

        if (resp.ok) {
          const ids = operations.map(op => op.id);
          await BcNative.dbExecute(
            `UPDATE _off_outbox SET status = 'SYNCED' WHERE id IN (${ids.map(() => '?').join(',')})`,
            ids,
          );
          synced += operations.length;
        } else {
          for (const op of operations) {
            const retries = (op.retry_count as number) + 1;
            const newStatus = retries >= 5 ? 'DEAD' : 'PENDING';
            await BcNative.dbExecute(
              `UPDATE _off_outbox SET retry_count = ?, status = ? WHERE id = ?`,
              [retries, newStatus, op.id],
            );
          }
          errors += operations.length;
        }
      } catch {
        for (const op of operations) {
          const retries = (op.retry_count as number) + 1;
          const newStatus = retries >= 5 ? 'DEAD' : 'PENDING';
          await BcNative.dbExecute(
            `UPDATE _off_outbox SET retry_count = ?, status = ? WHERE id = ?`,
            [retries, newStatus, op.id],
          );
        }
        errors += operations.length;
      }
    }

    return { synced, errors };
  },

  async syncPull(baseUrl?: string): Promise<{ applied: number }> {
    const url = baseUrl || BcSetup.getConfig().baseUrl;
    const headers = buildHeaders();
    let applied = 0;

    const stateRows = await BcNative.dbSelect(
      'SELECT last_pull_version FROM _off_sync_state LIMIT 1',
      [],
    );
    const sinceVersion = (stateRows[0] as Record<string, number>)?.last_pull_version ?? 0;

    try {
      const resp = await fetch(
        `${url}/api/v1/sync/pull?since_version=${sinceVersion}&device_id=${encodeURIComponent(_deviceId)}`,
        { headers },
      );
      if (!resp.ok) return { applied: 0 };

      const body = await resp.json() as {
        changes: Array<{
          table_name: string;
          record_id: string;
          operation: string;
          data: Record<string, unknown>;
          version: number;
        }>;
        max_version: number;
      };

      if (!body.changes || body.changes.length === 0) return { applied: 0 };

      await BcNative.dbExecute('BEGIN TRANSACTION', []);
      try {
        for (const change of body.changes) {
          assertSafeTable(change.table_name);
          switch (change.operation) {
            case 'CREATE': {
              const keys = Object.keys(change.data);
              for (const k of keys) assertValidColumn(change.table_name, k);
              const placeholders = keys.map(() => '?');
              const vals = keys.map(k => change.data[k]);
              await BcNative.dbExecute(
                `INSERT OR REPLACE INTO ${change.table_name} (${keys.join(', ')}) VALUES (${placeholders.join(', ')})`,
                vals,
              );
              break;
            }
            case 'UPDATE': {
              const entries = Object.entries(change.data).filter(([k]) => k !== 'id');
              for (const [k] of entries) assertValidColumn(change.table_name, k);
              const setParts = entries.map(([k]) => `${k} = ?`);
              const vals = entries.map(([, v]) => v);
              vals.push(change.record_id);
              await BcNative.dbExecute(
                `UPDATE ${change.table_name} SET ${setParts.join(', ')} WHERE id = ?`,
                vals,
              );
              break;
            }
            case 'DELETE': {
              await BcNative.dbExecute(
                `UPDATE ${change.table_name} SET _off_deleted = 1 WHERE id = ?`,
                [change.record_id],
              );
              break;
            }
          }
          applied++;
        }

        await BcNative.dbExecute(
          'UPDATE _off_sync_state SET last_pull_version = ?, last_sync_at = ?',
          [body.max_version, new Date().toISOString()],
        );

        await BcNative.dbExecute('COMMIT', []);
      } catch (err) {
        await BcNative.dbExecute('ROLLBACK', []);
        throw err;
      }
    } catch { /* */ }

    return { applied };
  },
};

async function fetchFromServer(model: string, params?: FetchParams): Promise<FetchResult> {
  const baseUrl = BcSetup.getConfig().baseUrl;
  const headers = buildHeaders();

  let url = `${baseUrl}/api/v1/${model}`;
  const qp: string[] = [];
  if (params?.page) qp.push(`page=${params.page}`);
  if (params?.pageSize) qp.push(`page_size=${params.pageSize}`);
  if (params?.search) qp.push(`q=${encodeURIComponent(params.search)}`);
  if (params?.sort && params.sort.length > 0) {
    qp.push(`sort=${params.sort[0].field}`);
    qp.push(`order=${params.sort[0].direction}`);
  }
  if (params?.filters) {
    for (const [key, val] of Object.entries(params.filters)) {
      qp.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(val))}`);
    }
  }
  if (qp.length > 0) url += `?${qp.join('&')}`;

  const resp = await fetch(url, { headers });
  const body = await resp.json();
  return normalizeResponse(body);
}

async function fetchOneFromServer(model: string, id: string): Promise<Record<string, unknown> | null> {
  const baseUrl = BcSetup.getConfig().baseUrl;
  const headers = buildHeaders();

  const resp = await fetch(`${baseUrl}/api/v1/${model}/${id}`, { headers });
  if (!resp.ok) return null;
  return resp.json();
}

async function saveToServer(model: string, data: Record<string, unknown>, method: 'POST' | 'PUT', id?: string): Promise<SaveResult> {
  const baseUrl = BcSetup.getConfig().baseUrl;
  const headers = { ...buildHeaders(), 'Content-Type': 'application/json' };

  const url = id ? `${baseUrl}/api/v1/${model}/${id}` : `${baseUrl}/api/v1/${model}`;
  const resp = await fetch(url, { method, headers, body: JSON.stringify(data) });
  const body = await resp.json();
  return { id: body.id || id || '', success: resp.ok };
}

async function deleteFromServer(model: string, id: string): Promise<DeleteResult> {
  const baseUrl = BcSetup.getConfig().baseUrl;
  const headers = buildHeaders();

  const resp = await fetch(`${baseUrl}/api/v1/${model}/${id}`, { method: 'DELETE', headers });
  return { success: resp.ok };
}
