import React, {useCallback, useEffect, useState} from "react";
import {defineMessages, useIntl} from "react-intl";
import {Center, Heading, Stack} from "@chakra-ui/core";
import {DEFAULT_LIMIT, DEFAULT_PAGE} from "src/constants";
import {useApi, useUser} from "src/hooks";
import AuthTokenTable from "../AuthTokenTable";

const messages = defineMessages({
  tableTitle: {
    id: "auth_tokens.table.title",
    defaultMessage: "Auth tokens: {tokens}/{total} | Page {page}/{maxPage}",
  },
});

function Layout() {
  const intl = useIntl();
  const fetchJSON = useApi();

  const [user] = useUser();
  const [page, setPage] = useState(DEFAULT_PAGE);
  const [limit, setLimit] = useState(DEFAULT_LIMIT);
  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [tokens, setTokens] = useState<Array<string>>([]);

  const fetchAuthTokens = useCallback(
    async ({page, limit}: { page: number; limit: number }) => {
      if (!user) {
        return null;
      }

      return await fetchJSON(`/auth/v1/users/${user.id}/tokens`, "GET", new URLSearchParams({
        page: String(page),
        limit: String(limit),
      }));
    },
    [fetchJSON, user]
  );

  useEffect(() => {
    let mounted = true
    if (!user) {
      setIsLoading(false);
      return;
    }

    const load = async () => {
      try {
        const response = await fetchAuthTokens({page, limit});
        if (!mounted) {
          return
        }

        setIsLoading(false);
        setTokens(response.auth_tokens || []);
        setTotal(response.total || 0);
      } catch (err) {
        if (!mounted) {
          return
        }

        setIsLoading(false);
      }
    };

    load();

    return function cleanup() {
      mounted = false
    }
  }, [user, page, limit, fetchAuthTokens]);

  if (!user) {
    return null;
  }

  return (
    <Stack flex={1}>
      <Heading m={4}>
        <Center>
          {intl.formatMessage(messages.tableTitle, {
            tokens: tokens.length,
            maxPage: Math.ceil(total / limit),
            total: total,
            page,
          })}
        </Center>
      </Heading>
      <AuthTokenTable
        isLoaded={!isLoading}
        tokens={tokens}
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
