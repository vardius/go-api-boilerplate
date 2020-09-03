import React, {createContext, ReactChild} from "react";
import {IntlProvider} from "react-intl";

type IntlProviderProps = React.ComponentProps<typeof IntlProvider>;

export const MessagesContext = createContext<[
  IntlProviderProps["messages"],
  React.Dispatch<React.SetStateAction<IntlProviderProps["messages"]>>
]>([undefined, () => {
}]);

export interface Props {
  children: ReactChild;
}

const MessagesContextProvider = ({children}: Props) => {
  const state = React.useState(undefined as IntlProviderProps["messages"]);

  return (
    <MessagesContext.Provider value={state}>
      {children}
    </MessagesContext.Provider>
  );
};

export default MessagesContextProvider;
