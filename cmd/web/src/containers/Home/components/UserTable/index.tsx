import React from "react";
import {defineMessages, useIntl} from "react-intl";
import {Box, Skeleton,} from "@chakra-ui/core";
import {User} from "src/types";
import {PaginatedTable} from "src/components/common"

const messages = defineMessages({
  id: {
    id: "user_table.header.id",
    defaultMessage: "ID",
  },
  email: {
    id: "user_table.header.email",
    defaultMessage: "Email",
  },
});

export interface Props {
  isLoaded: boolean;
  users: Array<User>;
  limit: number;
  page: number;
  total: number;
  onPageChange?: (v: number) => void;
  onLimitChange?: (v: number) => void;
}

const UserTable = (props: Props) => {
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
            {intl.formatMessage(messages.email)}
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
            <Box as="th">
              <Skeleton height="30px"/>
            </Box>
          </Box>
        ))}
        {props.isLoaded &&
        props.users.map((user, idx) => (
          <Box as="tr" key={user.id}>
            <Box as="th" scope="row">{idx}</Box>
            <Box as="th">{user.id}</Box>
            <Box as="th">{user.email}</Box>
          </Box>
        ))}
      </Box>
    </PaginatedTable>
  );
};

export default UserTable;
