import React, {lazy, Suspense} from "react";
import {Redirect, Route, Switch} from "react-router-dom";
import {Center, Flex, Spinner} from "@chakra-ui/core";
import ErrorBoundary from "src/components/common/ErrorBoundary";
import getPath from "src/routes";
import {useQuery, useUser} from "src/hooks";

import styles from "./Layout.module.scss";
import Header from "../Header";
import Footer from "../Footer";
import {RouteProps} from "react-router";

const Home = lazy(() => import("src/containers/Home"));
const AuthTokens = lazy(() => import("src/containers/AuthTokens"));
const ClientCredentials = lazy(() => import("src/containers/ClientCredentials"));
const NotFound = lazy(() => import("src/containers/NotFound"));
const Authorize = lazy(() => import("src/containers/Authorize"));
const Login = lazy(() => import("src/containers/Login"));

const PrivateRoute = ({component, children, ...rest}: RouteProps) => {
  const [user] = useUser();

  return (
    <Route
      {...rest}
      render={({location}) =>
        user ? (
          children
        ) : (
          <Redirect
            to={{
              pathname: "/login",
              state: {from: location}
            }}
          />
        )
      }
    />
  );
};

function Layout() {
  const query = useQuery();
  const redirectPath = query.get("r");

  if (redirectPath) {
    return <Redirect to={redirectPath}/>
  }

  return (
    <div className={styles.site}>
      <ErrorBoundary>
        <Flex minHeight="100vh" flexDirection="column">
          <Header/>
          <Suspense
            fallback={
              <Center>
                <Spinner thickness="4px" speed="0.65s" size="xl"/>
              </Center>
            }
          >
            <Switch>
              <Route exact path={getPath("home")} component={Home}/>
              <Route exact path={getPath("login")} component={Login}/>
              <PrivateRoute exact path={getPath("auth_tokens")} component={AuthTokens}/>
              <PrivateRoute exact path={getPath("client_credentials")} component={ClientCredentials}/>
              <PrivateRoute exact path={getPath("client_authorize")} component={Authorize}/>
              <Route component={NotFound}/>
            </Switch>
          </Suspense>
          <Footer/>
        </Flex>
      </ErrorBoundary>
    </div>
  );
}

export default Layout;
