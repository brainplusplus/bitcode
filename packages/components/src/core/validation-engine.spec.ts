import { validateBuiltIn, runValidationPipeline } from './validation-engine';
import { BcSetup } from './bc-setup';

describe('validateBuiltIn', () => {
  beforeEach(() => { BcSetup.reset(); });

  it('passes when no rules', () => {
    const result = validateBuiltIn('hello', {});
    expect(result.valid).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('fails required on empty string', () => {
    const result = validateBuiltIn('', { required: true });
    expect(result.valid).toBe(false);
    expect(result.errors.length).toBeGreaterThan(0);
  });

  it('passes required on non-empty', () => {
    const result = validateBuiltIn('hello', { required: true });
    expect(result.valid).toBe(true);
  });

  it('fails minLength', () => {
    const result = validateBuiltIn('ab', { minLength: 5 });
    expect(result.valid).toBe(false);
  });

  it('passes minLength', () => {
    const result = validateBuiltIn('hello world', { minLength: 5 });
    expect(result.valid).toBe(true);
  });

  it('fails maxLength', () => {
    const result = validateBuiltIn('hello world', { maxLength: 5 });
    expect(result.valid).toBe(false);
  });

  it('fails pattern', () => {
    const result = validateBuiltIn('abc123', { pattern: '^[a-zA-Z]+$' });
    expect(result.valid).toBe(false);
  });

  it('passes pattern', () => {
    const result = validateBuiltIn('hello', { pattern: '^[a-zA-Z]+$' });
    expect(result.valid).toBe(true);
  });

  it('uses custom patternMessage', () => {
    const result = validateBuiltIn('123', { pattern: '^[a-zA-Z]+$', patternMessage: 'Letters only!' });
    expect(result.errors[0]).toBe('Letters only!');
  });

  it('skips validation on empty non-required', () => {
    const result = validateBuiltIn('', { minLength: 5, maxLength: 10, pattern: '^[a-z]+$' });
    expect(result.valid).toBe(true);
  });
});

describe('runValidationPipeline', () => {
  beforeEach(() => { BcSetup.reset(); });

  it('runs built-in then stops on failure', async () => {
    const customCalled = jest.fn();
    const result = await runValidationPipeline({
      value: '',
      builtIn: { required: true },
      customValidator: customCalled,
    });
    expect(result.valid).toBe(false);
    expect(customCalled).not.toHaveBeenCalled();
  });

  it('runs custom validator after built-in passes', async () => {
    const result = await runValidationPipeline({
      value: 'test@competitor.com',
      builtIn: { required: true },
      customValidator: async (v) => String(v).endsWith('@competitor.com') ? 'Blocked' : null,
    });
    expect(result.valid).toBe(false);
    expect(result.errors[0]).toBe('Blocked');
  });

  it('passes when all levels pass', async () => {
    const result = await runValidationPipeline({
      value: 'hello',
      builtIn: { required: true, minLength: 3 },
      customValidator: async () => null,
    });
    expect(result.valid).toBe(true);
  });
});
