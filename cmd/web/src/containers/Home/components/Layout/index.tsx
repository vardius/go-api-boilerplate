import React, {useCallback, useEffect, useState} from "react";
import {defineMessages, useIntl} from "react-intl";
import {Center, Heading, Stack} from "@chakra-ui/core";
import {DEFAULT_LIMIT, DEFAULT_PAGE} from "src/constants";
import {useApi} from "src/hooks";
import UserTable from "../UserTable";

const messages = defineMessages({
  tableTitle: {
    id: "home.table.title",
    defaultMessage: "User: {users}/{total} | Page {page}/{maxPage}",
  },
});

function Layout() {
  const intl = useIntl();
  const fetchJSON = useApi();

  const [page, setPage] = useState(DEFAULT_PAGE);
  const [limit, setLimit] = useState(DEFAULT_LIMIT);
  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [users, setUsers] = useState([]);

  const fetchUsers = useCallback(
    async ({ page, limit }: { page: number; limit: number }) => {
      return await fetchJSON("/users/v1", "GET", {
        page: String(page),
        limit: String(limit),
      });
    },
    [fetchJSON]
  );

  useEffect(() => {
    const load = async () => {
      try {
        const response = await fetchUsers({ page, limit });

        setIsLoading(false);
        setUsers(response.users || []);
        setTotal(response.total || 0);
      } catch (err) {
        console.error(err);
      }
    };

    load();
  }, [page, limit, fetchUsers]);

  return (
    <Stack flex={1}>
      <Heading m={4}>
        <Center>
          {intl.formatMessage(messages.tableTitle, {
            users: users.length,
            maxPage: Math.ceil(total / limit),
            total: total,
            page,
          })}
        </Center>
      </Heading>
      <UserTable
        isLoaded={!isLoading}
        users={users}
        page={page}
        limit={limit}
        total={total}
        onPageChange={(newPage: number) => setPage(newPage)}
        onLimitChange={(newLimit: number) => setLimit(newLimit)}
      />
    </Stack>
  );
}

export default Layout;
