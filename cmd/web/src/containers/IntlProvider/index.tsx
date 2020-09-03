import React from "react";
import LanguageProvider, {LanguageProps} from "./components/LanguageProvider";
import MessagesContextProvider from "./components/MessagesContext";

export type IntlProps = LanguageProps;

const Intl = ({children, ...attributes}: IntlProps) => (
  <MessagesContextProvider>
    <LanguageProvider {...attributes}>
      {React.Children.only(children)}
    </LanguageProvider>
  </MessagesContextProvider>
);

export default Intl;
