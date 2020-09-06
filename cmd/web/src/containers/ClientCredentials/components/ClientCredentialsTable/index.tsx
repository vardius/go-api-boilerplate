import React from "react";
import {defineMessages, useIntl} from "react-intl";
import {Box, Skeleton,} from "@chakra-ui/core";
import {Client} from "src/types";
import {PaginatedTable} from "src/components/common"

const messages = defineMessages({
  id: {
    id: "client_credentials_table.header.id",
    defaultMessage: "ID",
  },
  secret: {
    id: "client_credentials_table.header.secret",
    defaultMessage: "Secret",
  },
  domain: {
    id: "client_credentials_table.header.domain",
    defaultMessage: "Domain",
  },
});

export interface Props {
  isLoaded: boolean;
  clients: Array<Client>;
  limit: number;
  page: number;
  total: number;
  onPageChange?: (v: number) => void;
  onLimitChange?: (v: number) => void;
}

const ClientCredentialsTable = (props: Props) => {
  const intl = useIntl();

  return (
    <PaginatedTable isLoaded={props.isLoaded} limit={props.limit} page={props.page} total={props.total}
                    onPageChange={props.onPageChange} onLimitChange={props.onLimitChange}>
      <Box as="thead">
        <Box as="tr">
          <Box as="th" scope="col">
            #
          </Box>
          <Box as="th" scope="col">
            {intl.formatMessage(messages.id)}
          </Box>
          <Box as="th" scope="col">
            {intl.formatMessage(messages.secret)}
          </Box>
          <Box as="th" scope="col">
            {intl.formatMessage(messages.domain)}
          </Box>
        </Box>
      </Box>
      <Box as="tbody">
        {!props.isLoaded &&
        [...Array(props.limit)].map((x, i) => (
          <Box as="tr" key={i}>
            <Box as="th" scope="row">
              <Skeleton height="30px"/>
            </Box>
            <Box as="th">
              <Skeleton height="30px"/>
            </Box>
          </Box>
        ))}
        {props.isLoaded &&
        props.clients.map((client, idx) => (
          <Box as="tr" key={idx}>
            <Box as="th" scope="row">
              {idx}
            </Box>
            <Box as="th">{client.id}</Box>
            <Box as="th">{client.secret}</Box>
            <Box as="th">{client.domain}</Box>
          </Box>
        ))}
      </Box>
    </PaginatedTable>
  );
};

export default ClientCredentialsTable;
