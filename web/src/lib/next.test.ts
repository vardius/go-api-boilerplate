import { describe, expect, it } from 'vitest';
import { isServerPath, safeNext, signInPath } from './next';

const origin = 'http://localhost:8080';

describe('safeNext', () => {
  it.each([
    ['/apps', '/apps'],
    ['/consent?request=abc#top', '/consent?request=abc#top'],
    ['/api/auth/oauth/authorize?client_id=x&scope=user%3Aread', '/api/auth/oauth/authorize?client_id=x&scope=user%3Aread'],
    // The auth service may send its authorize URL absolute, built from its public issuer URL.
    [`${origin}/api/auth/oauth/authorize?client_id=x`, '/api/auth/oauth/authorize?client_id=x'],
  ])('keeps the same-origin destination %s', (next, expected) => {
    expect(safeNext(next, origin)).toBe(expected);
  });

  it.each([
    ['missing', null],
    ['empty', ''],
    ['another site', 'https://evil.example/'],
    ['a protocol-relative URL', '//evil.example/path'],
    ['a backslash trick', '/\\evil.example'],
    ['a tab trick', '/\t/evil.example'],
    ['a dot segment before a double slash', '/.//evil.example'],
    ['a parent segment before a double slash', '/a/..//evil.example'],
    ['an encoded dot segment', '/%2e//evil.example'],
    ['a javascript: URL', 'javascript:alert(1)'],
    ['a relative path without a slash', 'profile'],
    ['a look-alike host', 'http://localhost:8080.evil.example/'],
    ['another port', 'http://localhost:9090/apps'],
    ['another scheme', 'https://localhost:8080/apps'],
  ])('drops %s', (_, next) => {
    expect(safeNext(next, origin)).toBeUndefined();
  });
});

describe('isServerPath', () => {
  it('tells API routes from app routes', () => {
    expect(isServerPath('/api/auth/oauth/authorize?x=1')).toBe(true);
    expect(isServerPath('/apps')).toBe(false);
  });
});

describe('signInPath', () => {
  it('carries the destination as an encoded query parameter', () => {
    expect(signInPath('/consent?request=a&b')).toBe('/sign-in?next=%2Fconsent%3Frequest%3Da%26b');
    expect(signInPath()).toBe('/sign-in');
  });
});
