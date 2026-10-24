import { describe, expect, it } from 'vitest';
import { describeUserAgent } from './device';

describe('describeUserAgent', () => {
  it.each([
    [
      'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36',
      'Chrome',
      'macOS',
    ],
    [
      'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0',
      'Edge',
      'Windows',
    ],
    [
      'Mozilla/5.0 (iPhone; CPU iPhone OS 19_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/19.0 Mobile/15E148 Safari/604.1',
      'Safari',
      'iOS',
    ],
    ['Mozilla/5.0 (X11; Linux x86_64; rv:144.0) Gecko/20100101 Firefox/144.0', 'Firefox', 'Linux'],
    [
      'Mozilla/5.0 (Linux; Android 16; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Mobile Safari/537.36',
      'Chrome',
      'Android',
    ],
  ])('names %s', (ua, browser, os) => {
    expect(describeUserAgent(ua)).toEqual({ browser, os });
  });

  it('leaves out what it cannot tell', () => {
    expect(describeUserAgent('curl/8.10.1')).toEqual({ browser: undefined, os: undefined });
    expect(describeUserAgent(undefined)).toEqual({ browser: undefined, os: undefined });
  });
});
