import {
  isArgumentElement,
  isDateElement,
  isNumberElement,
  isPluralElement,
  isSelectElement,
  isTagElement,
  isTimeElement,
  parse,
  type MessageFormatElement,
} from '@formatjs/icu-messageformat-parser';
import { describe, expect, it } from 'vitest';
import { locales, messages, pickLocale, type Locale } from './messages';

/** The argument names a message uses, and every plural it contains, by walking the parsed ICU tree. */
function inspect(elements: MessageFormatElement[], found = { args: new Set<string>(), plurals: [] as string[][] }) {
  for (const element of elements) {
    if (
      isArgumentElement(element) ||
      isNumberElement(element) ||
      isDateElement(element) ||
      isTimeElement(element)
    ) {
      found.args.add(element.value);
    } else if (isPluralElement(element) || isSelectElement(element)) {
      found.args.add(element.value);
      if (isPluralElement(element)) found.plurals.push(Object.keys(element.options));
      for (const option of Object.values(element.options)) inspect(option.value, found);
    } else if (isTagElement(element)) {
      inspect(element.children, found);
    }
  }
  return found;
}

const english = messages.en;

describe.each(locales.filter((locale) => locale !== 'en'))('%s messages', (locale: Locale) => {
  const translated = messages[locale];
  const categories = new Intl.PluralRules(locale).resolvedOptions().pluralCategories;

  it.each(Object.keys(english) as (keyof typeof english)[])('%s matches English', (id) => {
    const source = inspect(parse(english[id]));
    const target = inspect(parse(translated[id]));

    expect(translated[id].trim()).not.toBe('');
    expect([...target.args].sort()).toEqual([...source.args].sort());
    for (const plural of target.plurals) expect(plural).toEqual(expect.arrayContaining(categories));
  });
});

describe('pickLocale', () => {
  it('takes the first language the app speaks', () => {
    expect(pickLocale(['de-DE', 'pl-PL', 'en-GB'])).toBe('pl');
    expect(pickLocale(['EN-us'])).toBe('en');
  });

  it('falls back to English', () => {
    expect(pickLocale(['fr', 'de'])).toBe('en');
    expect(pickLocale([])).toBe('en');
  });
});
