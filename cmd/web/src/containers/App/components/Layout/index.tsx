import React, { lazy, Suspense } from "react";
import { Route, Switch } from "react-router-dom";
import { Spinner, Flex } from "@chakra-ui/core";
import getPath from "src/routes";

import styles from "./Layout.module.scss";
import ErrorBoundary from "../ErrorBoundary";
import Header from "../Header";
import Footer from "../Footer";

const Home = lazy(() => import("src/containers/Home"));
const NotFound = lazy(() => import("src/containers/NotFound"));

function Layout() {
  return (
    <div className={styles.site}>
      <ErrorBoundary>
        <Flex minHeight="100vh" flexDirection="column">
          <Header />
          <Suspense
            fallback={<Spinner thickness="4px" speed="0.65s" size="xl" />}
          >
            <Switch>
              <Route exact path={getPath("home")} component={Home} />
              <Route component={NotFound} />
            </Switch>
          </Suspense>
          <Footer />
        </Flex>
      </ErrorBoundary>
    </div>
  );
}

export default Layout;
