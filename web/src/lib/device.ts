export interface Device {
  browser: string | undefined;
  os: string | undefined;
}

// Order matters: Edge and Opera also say "Chrome", Chrome also says "Safari".
const browsers: readonly [RegExp, string][] = [
  [/\bEdg(e|A|iOS)?\//, 'Edge'],
  [/\b(OPR|Opera)\//, 'Opera'],
  [/\b(Firefox|FxiOS)\//, 'Firefox'],
  [/\b(Chrome|CriOS|Chromium)\//, 'Chrome'],
  [/\bVersion\/[\d.]+.*\bSafari\//, 'Safari'],
];

// iPhones and iPads say "like Mac OS X", and Android says "Linux".
const systems: readonly [RegExp, string][] = [
  [/\b(iPhone|iPad|iPod)\b/, 'iOS'],
  [/\bAndroid\b/, 'Android'],
  [/\bCrOS\b/, 'ChromeOS'],
  [/\bWindows\b/, 'Windows'],
  [/\bMac OS X\b|\bMacintosh\b/, 'macOS'],
  [/\bLinux\b/, 'Linux'],
];

/** A rough "which browser on which system" from a User-Agent string, for telling sessions apart. */
export function describeUserAgent(userAgent: string | null | undefined): Device {
  const ua = userAgent ?? '';
  return {
    browser: browsers.find(([pattern]) => pattern.test(ua))?.[1],
    os: systems.find(([pattern]) => pattern.test(ua))?.[1],
  };
}
