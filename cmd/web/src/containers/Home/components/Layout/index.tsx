import React, { useEffect, useState } from "react";
import { defineMessages, useIntl } from "react-intl";
import { SimpleGrid, Text, Box, Stack } from "@chakra-ui/core";
import { User } from "src/types";
import { DEFAULT_PAGE, DEFAULT_LIMIT } from "src/constants";
import { fetchJSON } from "src/api";

const messages = defineMessages({
  tableTitle: {
    id: "home.table.title",
    defaultMessage: "{users}/{total} Users | Page {page}/{maxPage}",
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

  const [page] = useState(DEFAULT_PAGE);
  const [limit] = useState(DEFAULT_LIMIT);
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
      <Box>
        {intl.formatMessage(messages.tableTitle, {
          users: users.length,
          maxPage: Math.ceil(total / limit),
          total,
          page,
        })}
      </Box>
      <SimpleGrid columns={1} spacing={10}>
        {users.map((user: User) => (
          <Text key={user.id}>
            {user.id} - {user.email}
          </Text>
        ))}
      </SimpleGrid>
    </Stack>
  );
}

export default Layout;
