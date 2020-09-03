import React, {ReactChild, useEffect} from "react";
import {IntlProvider} from "react-intl";
import {DEFAULT_LOCALE} from "src/constants";
import {loadTranslation} from "src/intl/i18n";
import {useLocale} from "src/hooks";
import useMessages from "../../useMessages";

export interface LanguageProps {
  children: ReactChild;
}

export const flattenMessages = (
  nestedMessages: object,
  prefix = ""
): Record<string, string> => {
  if (nestedMessages === null) {
    return {};
  }

  return Object.keys(nestedMessages).reduce((messages, key) => {
    // @ts-ignore
    const value = nestedMessages[key];
    const prefixedKey = prefix ? `${prefix}.${key}` : key;

    if (typeof value === "string") {
      Object.assign(messages, {[prefixedKey]: value});
    } else {
      Object.assign(messages, flattenMessages(value, prefixedKey));
    }

    return messages;
  }, {});
};

function LanguageProvider({children}: LanguageProps) {
  const [locale] = useLocale();
  const [messages, setMessages] = useMessages();

  useEffect(() => {
    // @ts-ignore
    loadTranslation(locale).then((messages: object) => {
      setMessages(flattenMessages(messages));
    });
  }, [locale, setMessages]);

  return (
    <IntlProvider
      locale={locale}
      messages={messages}
      defaultLocale={DEFAULT_LOCALE}
    >
      {React.Children.only(children)}
    </IntlProvider>
  );
}

export default LanguageProvider;
