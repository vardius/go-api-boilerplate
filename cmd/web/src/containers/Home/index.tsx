import React, {useCallback, useEffect, useState} from "react";
import {defineMessages, useIntl} from "react-intl";
import {Center, Heading, Stack} from "@chakra-ui/core";
import {DEFAULT_LIMIT, DEFAULT_PAGE} from "src/constants";
import {User} from "src/types";
import {useApi} from "src/hooks";
import UserTable from "./components/UserTable";

const messages = defineMessages({
  tableTitle: {
    id: "home.table.title",
    defaultMessage: "User: {users}/{total} | Page {page}/{maxPage}",
  },
  signedOut: {
    id: "home.signed_out",
    defaultMessage: "Sign in to see your account.",
  },
});

function Home() {
  const intl = useIntl();
  const fetchJSON = useApi("users");

  const [page, setPage] = useState(DEFAULT_PAGE);
  const [limit, setLimit] = useState(DEFAULT_LIMIT);
  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [users, setUsers] = useState<Array<User>>([]);
  const [failed, setFailed] = useState(false);

  const fetchUsers = useCallback(
    async ({page, limit}: { page: number; limit: number }) => {
      return await fetchJSON("/", "GET", new URLSearchParams({
        page: String(page),
        limit: String(limit),
      }));
    },
    [fetchJSON]
  );

  useEffect(() => {
    let mounted = true
    const load = async () => {
      try {
        const response = await fetchUsers({page, limit});
        if (!mounted) {
          return
        }

        setIsLoading(false);
        setFailed(false);
        setUsers(response.users || []);
        setTotal(response.total || 0);
      } catch (err) {
        if (!mounted) {
          return
        }

        setIsLoading(false);
        setFailed(true);
      }
    };

    load();

    return function cleanup() {
      mounted = false
    }
  }, [page, limit, fetchUsers]);

  if (failed) {
    return (
      <Stack flex={1}>
        <Heading m={4}>
          <Center>{intl.formatMessage(messages.signedOut)}</Center>
        </Heading>
      </Stack>
    );
  }

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

export default Home;
