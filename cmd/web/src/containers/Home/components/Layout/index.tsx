import React, { useEffect, useState } from "react";
import { defineMessages, useIntl } from "react-intl";
import { Heading, Stack } from "@chakra-ui/core";
import { DEFAULT_PAGE, DEFAULT_LIMIT } from "src/constants";
import { fetchJSON } from "src/api";
import UserTable from "../UserTable";

const messages = defineMessages({
  tableTitle: {
    id: "home.table.title",
    defaultMessage: "User: {users}/{total} | Page {page}/{maxPage}",
  },
});

const fetchUsers = async ({ page, limit }: { page: number; limit: number }) => {
  const json = await fetchJSON("/users/v1", "GET", {
    page: String(page),
    limit: String(limit),
  });

  return json;
};

function Layout() {
  const intl = useIntl();

  const [page, setPage] = useState(DEFAULT_PAGE);
  const [limit, setLimit] = useState(DEFAULT_LIMIT);
  const [total, setTotal] = useState(0);
  const [users, setUsers] = useState([]);

  useEffect(() => {
    const load = async () => {
      try {
        const response = await fetchUsers({ page, limit });

        setUsers(response.users || []);
        setTotal(response.total || 0);
      } catch (err) {
        console.error(err);
      }
    };

    load();
  }, [page, limit]);

  return (
    <Stack flex={1}>
      <Heading m={4}>
        {intl.formatMessage(messages.tableTitle, {
          users: users.length,
          maxPage: Math.ceil(total / limit),
          total: total,
          page,
        })}
      </Heading>
      <UserTable
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
