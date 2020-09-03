import React, {createContext, ReactChild} from "react";
import {LOCALE} from "src/types";
import {DEFAULT_LOCALE} from "src/constants";

export const LocaleContext = createContext<[LOCALE, React.Dispatch<React.SetStateAction<LOCALE>>]>([DEFAULT_LOCALE, () => {
}]);

export interface Props {
  children: ReactChild;
}

const LocaleContextProvider = (props: Props) => {
  const state = React.useState(DEFAULT_LOCALE);

  return (
    <LocaleContext.Provider value={state}>
      {props.children}
    </LocaleContext.Provider>
  );
};

export default LocaleContextProvider;
