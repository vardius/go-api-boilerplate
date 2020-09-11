import React from "react";
import ReactDOM from "react-dom";
import {BrowserRouter as Router} from "react-router-dom";
import {ChakraProvider} from "@chakra-ui/core";

import App from "src/containers/App";
import IntlProvider from "src/containers/IntlProvider";
import AuthProvider from "src/context/AuthContext";
import UserProvider from "src/context/UserContext";
import LocaleContextProvider from "src/context/LocaleContext";

import * as serviceWorker from "./serviceWorker";

import theme from "src/theme/theme";
import "src/theme/scss/styles.scss";

const MOUNT_NODE = document.getElementById("root") as HTMLElement;

const render = () =>
  ReactDOM.render(
    <LocaleContextProvider>
      <IntlProvider>
        <Router>
          <AuthProvider>
            <UserProvider>
              <ChakraProvider resetCSS theme={theme}>
                <App/>
              </ChakraProvider>
            </UserProvider>
          </AuthProvider>
        </Router>
      </IntlProvider>
    </LocaleContextProvider>,
    MOUNT_NODE
  );

render();

// If you want your app to work offline and load faster, you can change
// unregister() to register() below. Note this comes with some pitfalls.
// Learn more about service workers: https://bit.ly/CRA-PWA
serviceWorker.unregister();
