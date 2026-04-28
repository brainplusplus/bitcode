import { BcSetup } from './bc-setup';

describe('BcSetup', () => {
  beforeEach(() => { BcSetup.reset(); });

  it('has sensible defaults', () => {
    const config = BcSetup.getConfig();
    expect(config.baseUrl).toBe('');
    expect(config.auth.type).toBe('none');
    expect(config.validateOn).toBe('blur');
    expect(config.size).toBe('md');
    expect(config.locale).toBe('en');
    expect(config.theme).toBe('light');
  });

  it('merges config on configure()', () => {
    BcSetup.configure({ baseUrl: '/api' });
    BcSetup.configure({ size: 'lg' });
    const config = BcSetup.getConfig();
    expect(config.baseUrl).toBe('/api');
    expect(config.size).toBe('lg');
  });

  it('resolves bearer auth headers', () => {
    BcSetup.configure({ auth: { type: 'bearer', token: 'my-jwt' } });
    const headers = BcSetup.getHeaders();
    expect(headers['Authorization']).toBe('Bearer my-jwt');
  });

  it('resolves function-based token', () => {
    BcSetup.configure({ auth: { type: 'bearer', token: () => 'dynamic-token' } });
    const headers = BcSetup.getHeaders();
    expect(headers['Authorization']).toBe('Bearer dynamic-token');
  });

  it('resolves custom header auth', () => {
    BcSetup.configure({ auth: { type: 'header', headerName: 'X-API-Key', headerValue: 'key123' } });
    const headers = BcSetup.getHeaders();
    expect(headers['X-API-Key']).toBe('key123');
  });

  it('merges custom headers', () => {
    BcSetup.configure({ headers: { 'X-Tenant': 'company-a' } });
    BcSetup.configure({ headers: { 'Accept-Language': 'id' } });
    const headers = BcSetup.getHeaders();
    expect(headers['X-Tenant']).toBe('company-a');
    expect(headers['Accept-Language']).toBe('id');
  });

  it('returns validation messages with params', () => {
    const msg = BcSetup.getValidationMessage('minLength', 8);
    expect(msg).toBe('Minimum 8 characters');
  });

  it('switches to Indonesian messages on locale change', () => {
    BcSetup.configure({ locale: 'id' });
    const msg = BcSetup.getValidationMessage('required');
    expect(msg).toBe('Wajib diisi');
  });

  it('registers and retrieves custom validators', () => {
    const fn = (value: unknown) => value === 'bad' ? 'Not allowed' : null;
    BcSetup.registerValidator('no-bad', fn);
    expect(BcSetup.getValidator('no-bad')).toBe(fn);
  });

  it('registers and retrieves reactivity rules', () => {
    const handler = () => {};
    BcSetup.reactivity({ 'field1': handler });
    expect(BcSetup.hasReactivityRule('field1')).toBe(true);
    expect(BcSetup.getReactivityRule('field1')).toBe(handler);
    expect(BcSetup.hasReactivityRule('field2')).toBe(false);
  });

  it('resets to defaults', () => {
    BcSetup.configure({ baseUrl: '/api', locale: 'id' });
    BcSetup.registerValidator('test', () => null);
    BcSetup.reactivity({ 'x': () => {} });
    BcSetup.reset();
    expect(BcSetup.getConfig().baseUrl).toBe('');
    expect(BcSetup.getConfig().locale).toBe('en');
    expect(BcSetup.getValidator('test')).toBeUndefined();
    expect(BcSetup.hasReactivityRule('x')).toBe(false);
  });
});
