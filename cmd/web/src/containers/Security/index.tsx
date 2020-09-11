import React from "react";
import {defineMessages, useIntl} from "react-intl";
import {Tab, TabList, TabPanel, TabPanels, Tabs} from "@chakra-ui/core";
import ClientList from "./components/ClientList";
import AuthTokenList from "./components/AuthTokenList";

const messages = defineMessages({
  account_security: {
    id: "security.nav.account_security",
    defaultMessage: "Account security",
  },
  api_access: {
    id: "security.nav.api_access",
    defaultMessage: "API Access",
  },
  sessions: {
    id: "security.card.sessions",
    defaultMessage: "Session",
  },
});

function Security() {
  const intl = useIntl();

  return (
    <Tabs isFitted variant="enclosed">
      <TabList mb="1em">
        <Tab>{intl.formatMessage(messages.account_security)}</Tab>
        <Tab>{intl.formatMessage(messages.api_access)}</Tab>
      </TabList>
      <TabPanels>
        <TabPanel>
          <AuthTokenList/>
        </TabPanel>
        <TabPanel>
          <ClientList/>
        </TabPanel>
      </TabPanels>
    </Tabs>
  );
}

export default Security;
