import { beforeEach, describe, expect, it, vi } from 'vitest';
import { initialLocale, saveLocale } from './locale';

function browserLanguages(...languages: string[]) {
  vi.spyOn(Navigator.prototype, 'languages', 'get').mockReturnValue(languages);
}

function blockStorage() {
  const blocked = () => {
    throw new DOMException('The operation is insecure.', 'SecurityError');
  };
  vi.spyOn(Storage.prototype, 'getItem').mockImplementation(blocked);
  vi.spyOn(Storage.prototype, 'setItem').mockImplementation(blocked);
}

describe('initialLocale', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("keeps the reader's saved choice over their browser's languages", () => {
    browserLanguages('en-GB');
    saveLocale('pl');

    expect(initialLocale()).toBe('pl');
  });

  it("uses the browser's languages when the saved choice isn't one the app speaks", () => {
    browserLanguages('pl-PL', 'en');
    window.localStorage.setItem('locale', 'fr');

    expect(initialLocale()).toBe('pl');
  });

  it("works with storage blocked, and saving there doesn't throw", () => {
    browserLanguages('pl');
    blockStorage();

    expect(() => {
      saveLocale('en');
    }).not.toThrow();
    expect(initialLocale()).toBe('pl');
  });
});
