import React from "react";
import {defineMessages, useIntl} from "react-intl";
import {Box, Skeleton, Text} from "@chakra-ui/core";
import {PaginatedTable} from "src/components/common"

const messages = defineMessages({
  token: {
    id: "auth_token_table.header.auth_token",
    defaultMessage: "Auth token",
  },
});

export interface Props {
  isLoaded: boolean;
  tokens: Array<string>;
  limit: number;
  page: number;
  total: number;
  onPageChange?: (v: number) => void;
  onLimitChange?: (v: number) => void;
}

const AuthTokenTable = (props: Props) => {
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
            {intl.formatMessage(messages.token)}
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
        props.tokens.map((token, idx) => (
          <Box as="tr" key={idx}>
            <Box as="th" scope="row">
              {idx}
            </Box>
            <Box as="th" scope="row">
              <Text fontSize="sm">{token}</Text>
            </Box>
          </Box>
        ))}
      </Box>
    </PaginatedTable>
  );
};

export default AuthTokenTable;
