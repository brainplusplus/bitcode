import { resolveUrl, normalizeResponse, buildHeaders } from './data-fetcher';
import { BcSetup } from './bc-setup';

describe('resolveUrl', () => {
  it('replaces placeholders', () => {
    expect(resolveUrl('/api/cities?province={province}', { province: 'jabar' }))
      .toBe('/api/cities?province=jabar');
  });

  it('handles multiple placeholders', () => {
    expect(resolveUrl('/api/{model}/{id}', { model: 'user', id: '123' }))
      .toBe('/api/user/123');
  });

  it('encodes special characters', () => {
    expect(resolveUrl('/api?q={q}', { q: 'hello world' }))
      .toBe('/api?q=hello%20world');
  });

  it('replaces missing values with empty string', () => {
    expect(resolveUrl('/api?x={missing}', {}))
      .toBe('/api?x=');
  });
});

describe('normalizeResponse', () => {
  beforeEach(() => { BcSetup.reset(); });

  it('handles plain array', () => {
    const result = normalizeResponse([1, 2, 3]);
    expect(result.data).toEqual([1, 2, 3]);
    expect(result.total).toBe(3);
  });

  it('handles { data: [...] }', () => {
    const result = normalizeResponse({ data: [1, 2], total: 100 });
    expect(result.data).toEqual([1, 2]);
    expect(result.total).toBe(100);
  });

  it('handles { results: [...] }', () => {
    const result = normalizeResponse({ results: [1], total_count: 50 });
    expect(result.data).toEqual([1]);
    expect(result.total).toBe(50);
  });

  it('handles { items: [...] }', () => {
    const result = normalizeResponse({ items: [1, 2, 3], count: 30 });
    expect(result.data).toEqual([1, 2, 3]);
    expect(result.total).toBe(30);
  });

  it('returns empty for unknown format', () => {
    const result = normalizeResponse({ foo: 'bar' });
    expect(result.data).toEqual([]);
    expect(result.total).toBe(0);
  });

  it('uses responseTransformer when set', () => {
    BcSetup.configure({ responseTransformer: (r: any) => ({ data: r.payload, total: r.meta.count }) });
    const result = normalizeResponse({ payload: [1, 2], meta: { count: 99 } });
    expect(result.data).toEqual([1, 2]);
    expect(result.total).toBe(99);
  });
});

describe('buildHeaders', () => {
  beforeEach(() => { BcSetup.reset(); });

  it('includes BcSetup headers', () => {
    BcSetup.configure({ headers: { 'X-Tenant': 'test' }, auth: { type: 'bearer', token: 'jwt' } });
    const headers = buildHeaders();
    expect(headers['X-Tenant']).toBe('test');
    expect(headers['Authorization']).toBe('Bearer jwt');
  });

  it('merges custom headers', () => {
    BcSetup.configure({ headers: { 'X-A': '1' } });
    const headers = buildHeaders({ 'X-B': '2' });
    expect(headers['X-A']).toBe('1');
    expect(headers['X-B']).toBe('2');
  });

  it('parses JSON string headers', () => {
    const headers = buildHeaders('{"X-Custom":"value"}');
    expect(headers['X-Custom']).toBe('value');
  });
});
