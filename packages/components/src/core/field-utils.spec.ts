import { createFieldState, markDirty, markTouched, resetFieldState, getAriaAttrs, getFieldClasses } from './field-utils';

describe('FieldState', () => {
  it('creates initial state', () => {
    const state = createFieldState('hello');
    expect(state.dirty).toBe(false);
    expect(state.touched).toBe(false);
    expect(state.pristine).toBe(true);
    expect(state.initialValue).toBe('hello');
  });

  it('marks dirty when value changes', () => {
    const state = createFieldState('hello');
    const dirty = markDirty(state, 'world');
    expect(dirty.dirty).toBe(true);
    expect(dirty.pristine).toBe(false);
  });

  it('marks not dirty when value matches initial', () => {
    const state = createFieldState('hello');
    const dirty = markDirty(state, 'hello');
    expect(dirty.dirty).toBe(false);
    expect(dirty.pristine).toBe(true);
  });

  it('marks touched', () => {
    const state = createFieldState('');
    const touched = markTouched(state);
    expect(touched.touched).toBe(true);
  });

  it('resets state', () => {
    let state = createFieldState('hello');
    state = markDirty(state, 'world');
    state = markTouched(state);
    const reset = resetFieldState(state);
    expect(reset.dirty).toBe(false);
    expect(reset.touched).toBe(false);
    expect(reset.pristine).toBe(true);
    expect(reset.initialValue).toBe('hello');
  });

  it('resets with new initial value', () => {
    const state = createFieldState('old');
    const reset = resetFieldState(state, 'new');
    expect(reset.initialValue).toBe('new');
  });
});

describe('getAriaAttrs', () => {
  it('sets aria-required', () => {
    const attrs = getAriaAttrs({ required: true });
    expect(attrs['aria-required']).toBe('true');
  });

  it('sets aria-invalid on invalid status', () => {
    const attrs = getAriaAttrs({ validationStatus: 'invalid' });
    expect(attrs['aria-invalid']).toBe('true');
  });

  it('sets aria-describedby for hint', () => {
    const attrs = getAriaAttrs({ name: 'email', hint: 'Enter email' });
    expect(attrs['aria-describedby']).toBe('email-hint');
  });

  it('returns empty for no special state', () => {
    const attrs = getAriaAttrs({});
    expect(Object.keys(attrs).length).toBe(0);
  });
});

describe('getFieldClasses', () => {
  it('returns base class', () => {
    const classes = getFieldClasses({});
    expect(classes['bc-field']).toBe(true);
    expect(classes['bc-field-md']).toBe(true);
  });

  it('sets size class', () => {
    const classes = getFieldClasses({ size: 'lg' });
    expect(classes['bc-field-lg']).toBe(true);
    expect(classes['bc-field-md']).toBe(false);
  });

  it('sets validation class', () => {
    const classes = getFieldClasses({ validationStatus: 'invalid' });
    expect(classes['bc-field-invalid']).toBe(true);
  });

  it('sets dirty and touched', () => {
    const classes = getFieldClasses({ dirty: true, touched: true });
    expect(classes['bc-field-dirty']).toBe(true);
    expect(classes['bc-field-touched']).toBe(true);
  });
});
