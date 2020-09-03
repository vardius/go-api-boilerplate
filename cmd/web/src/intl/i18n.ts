import {DEFAULT_LOCALE} from "src/constants";
import {LOCALE} from "src/types";

export function loadTranslation(locale: LOCALE) {
  const l = LOCALE[locale] || DEFAULT_LOCALE;
  switch (l) {
    case LOCALE.pl:
      return import("./i18n/pl.json");
    case LOCALE.en:
    default:
      return import("./i18n/en.json");
  }
}
