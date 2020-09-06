import React, {useCallback, useEffect, useState} from "react";
import {defineMessages, useIntl} from "react-intl";
import {Center, Heading, Stack} from "@chakra-ui/core";
import {DEFAULT_LIMIT, DEFAULT_PAGE} from "src/constants";
import {Client} from "src/types";
import {useApi, useUser} from "src/hooks";
import ClientCredentialsTable from "../ClientCredentialsTable";

const messages = defineMessages({
  tableTitle: {
    id: "clients.table.title",
    defaultMessage: "Credentials: {clients}/{total} | Page {page}/{maxPage}",
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
  const [clients, setClients] = useState<Array<Client>>([]);

  const fetchClients = useCallback(
    async ({page, limit}: { page: number; limit: number }) => {
      if (!user) {
        return null;
      }

      return await fetchJSON(`/auth/v1/clients`, "GET", new URLSearchParams({
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
        const response = await fetchClients({page, limit});
        if (!mounted) {
          return
        }

        setIsLoading(false);
        setClients(response.clients || []);
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
  }, [user, page, limit, fetchClients]);

  if (!user) {
    return null;
  }

  return (
    <Stack flex={1}>
      <Heading m={4}>
        <Center>
          {intl.formatMessage(messages.tableTitle, {
            clients: clients.length,
            maxPage: Math.ceil(total / limit),
            total: total,
            page,
          })}
        </Center>
      </Heading>
      <ClientCredentialsTable
        isLoaded={!isLoading}
        clients={clients}
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
