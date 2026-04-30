import { describe, test, expect, jest } from '@jest/globals';

describe('payment', () => {
  test('mocks with jest', () => {
    const spy = jest.fn();
    spy('hello');
    expect(spy).toHaveBeenCalledWith('hello');
    jest.clearAllMocks();
  });
});
