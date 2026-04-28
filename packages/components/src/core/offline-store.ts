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

function buildSelectSQL(table: string, params?: FetchParams): { sql: string; values: unknown[] } {
  const values: unknown[] = [];
  let sql = `SELECT * FROM ${table}`;
  const clauses: string[] = ['_off_deleted = 0'];

  if (params?.filters) {
    for (const [key, val] of Object.entries(params.filters)) {
      if (val === null || val === undefined) continue;
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
    const orderParts = params.sort.map(s => `${s.field} ${s.direction.toUpperCase()}`);
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
  const values: unknown[] = [];
  let sql = `SELECT COUNT(*) as cnt FROM ${table}`;
  const clauses: string[] = ['_off_deleted = 0'];

  if (params?.filters) {
    for (const [key, val] of Object.entries(params.filters)) {
      if (val === null || val === undefined) continue;
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
  const keys = Object.keys(data);
  const placeholders = keys.map(() => '?');
  const values = keys.map(k => data[k]);
  return {
    sql: `INSERT INTO ${table} (${keys.join(', ')}) VALUES (${placeholders.join(', ')})`,
    values,
  };
}

function buildUpdateSQL(table: string, id: string, data: Record<string, unknown>): { sql: string; values: unknown[] } {
  const entries = Object.entries(data).filter(([k]) => k !== 'id');
  const setParts = entries.map(([k]) => `${k} = ?`);
  const values = entries.map(([, v]) => v);
  values.push(id);
  return {
    sql: `UPDATE ${table} SET ${setParts.join(', ')} WHERE id = ?`,
    values,
  };
}

async function recordOutbox(table: string, recordId: string, operation: 'CREATE' | 'UPDATE' | 'DELETE', payload: Record<string, unknown>): Promise<void> {
  const idempotencyKey = `${table}:${recordId}:${operation}:${Date.now()}`;
  await BcNative.dbExecute(
    `INSERT INTO _off_outbox (table_name, record_id, operation, payload, idempotency_key) VALUES (?, ?, ?, ?, ?)`,
    [table, recordId, operation, JSON.stringify(payload), idempotencyKey],
  );
}

const _tableMap = new Map<string, string>();

function resolveTable(model: string): string {
  return _tableMap.get(model) || model;
}

export const OfflineStore = {

  registerTableMap(models: Array<{ name: string; table_name: string }>): void {
    _tableMap.clear();
    for (const m of models) {
      _tableMap.set(m.name, m.table_name);
    }
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
    const id = (data.id as string) || generateUUIDv7();
    const now = new Date().toISOString();

    const record: Record<string, unknown> = {
      ...data,
      id,
      _off_status: 'PENDING',
      _off_deleted: 0,
      _off_created_at: now,
      _off_updated_at: now,
    };

    const { sql, values } = buildInsertSQL(table, record);
    await BcNative.dbExecute(sql, values);
    await recordOutbox(table, id, 'CREATE', record);

    return { id, success: true };
  },

  async update(model: string, id: string, data: Record<string, unknown>): Promise<SaveResult> {
    if (!BcSetup.isModelOffline(model)) {
      return saveToServer(model, data, 'PUT', id);
    }

    const table = resolveTable(model);
    const now = new Date().toISOString();

    const record: Record<string, unknown> = {
      ...data,
      _off_status: 'PENDING',
      _off_updated_at: now,
    };

    const { sql, values } = buildUpdateSQL(table, id, record);
    await BcNative.dbExecute(sql, values);
    await recordOutbox(table, id, 'UPDATE', record);

    return { id, success: true };
  },

  async delete(model: string, id: string): Promise<DeleteResult> {
    if (!BcSetup.isModelOffline(model)) {
      return deleteFromServer(model, id);
    }

    const table = resolveTable(model);
    const now = new Date().toISOString();

    await BcNative.dbExecute(
      `UPDATE ${table} SET _off_deleted = 1, _off_status = 'PENDING', _off_updated_at = ? WHERE id = ?`,
      [now, id],
    );
    await recordOutbox(table, id, 'DELETE', { id });

    return { success: true };
  },

  async initFromServer(baseUrl?: string): Promise<void> {
    const url = baseUrl || BcSetup.getConfig().baseUrl;
    const headers = buildHeaders();

    try {
      const resp = await fetch(`${url}/api/v1/sync/schema`, { headers });
      if (!resp.ok) return;

      const body = await resp.json();
      const models = body.models as Array<{ name: string; table_name: string }> | undefined;
      if (!models || models.length === 0) return;

      BcSetup.registerOfflineModels(models.map(m => m.name));
      OfflineStore.registerTableMap(models);
    } catch {
      // Server unreachable — keep existing offline model config
    }
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
