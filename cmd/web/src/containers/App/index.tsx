import React, {lazy, ReactNode, Suspense} from "react";
import {Redirect, Route, Switch} from "react-router-dom";
import {Center, Flex, Spinner} from "@chakra-ui/core";
import ErrorBoundary from "src/components/common/ErrorBoundary";
import getPath from "src/routes";
import {useQuery, useUser} from "src/hooks";

import styles from "./App.module.scss";
import {RouteProps} from "react-router";
import Header from "./components/Header";
import Footer from "./components/Footer";

const Home = lazy(() => import("src/containers/Home"));
const Login = lazy(() => import("src/containers/Login"));
const Authorize = lazy(() => import("src/containers/Authorize"));
const Security = lazy(() => import("src/containers/Security"));
const NotFound = lazy(() => import("src/containers/NotFound"));

const PrivateRoute = ({component, children, ...rest}: RouteProps) => {
  const [user] = useUser();

  if (!user) {
    return (
      <Route
        {...rest}
        render={({location}) =>
          <Redirect
            to={{
              pathname: getPath("login"),
              state: {from: location}
            }}
          />
        }
      />
    );
  }

  return (
    <Route component={component} {...rest} />
  );
};

const Page = ({children}: {
  children: ReactNode;
}) => {
  return (
    <>
      <Header/>
      {children}
      <Footer/>
    </>
  );
};

function App() {
  const query = useQuery();
  const redirectPath = query.get("r");

  if (redirectPath) {
    return <Redirect to={redirectPath}/>
  }

  return (
    <div className={styles.site}>
      <ErrorBoundary>
        <Flex minHeight="100vh" flexDirection="column">
          <Suspense
            fallback={
              <Center minHeight="100vh">
                <Spinner thickness="4px" speed="0.65s" size="xl"/>
              </Center>
            }
          >
            <Switch>
              <Route exact path={getPath("home")} render={() => <Page><Home/></Page>}/>
              <Route exact path={getPath("login")} component={Login}/>
              <PrivateRoute exact path={getPath("security")} render={() => <Page><Security/></Page>}/>
              <PrivateRoute exact path={getPath("client_authorize")} component={Authorize}/>
              <Route component={NotFound}/>
            </Switch>
          </Suspense>
        </Flex>
      </ErrorBoundary>
    </div>
  );
}

export default App;
