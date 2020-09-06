import React, {useCallback, useState} from "react";
import {defineMessages, useIntl} from "react-intl";
import {Box, Button, Center, CircularProgress, Heading, Stack} from "@chakra-ui/core";
import {useApi, useQuery, useUser} from "src/hooks";
import {SubmitMessage} from "src/components/common";

const messages = defineMessages({
  title: {
    id: "authorize.title",
    defaultMessage: "Authorize",
  },
  submit: {
    id: "authorize.submit",
    defaultMessage: "Allow",
  },
  error: {
    id: "authorize.form.error",
    defaultMessage: "Authorization failed: {error}",
  },
});

function Authorize() {
  const intl = useIntl();
  const fetchJSON = useApi();
  const query = useQuery();
  const [user] = useUser();
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const authorize = useCallback(
    async () => {
      if (!user) {
        return null;
      }

      const response = await fetchJSON(`/auth/v1/authorize`, "GET", query);

      window.location.assign(response.location);
    },
    [fetchJSON, user, query]
  );

  if (!user) {
    return null;
  }

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    setIsLoading(true);

    try {
      await authorize();
    } catch (err) {
      setError(intl.formatMessage(messages.error, {error: err.message}));
    }

    setIsLoading(false);
  };

  return (
    <Stack flex={1}>
      <Heading m={4}>
        <Center>
          {intl.formatMessage(messages.title)}
        </Center>
      </Heading>
      <Box my={4} textAlign="left">
        <form onSubmit={handleSubmit}>
          {error && <SubmitMessage message={error} status="error"/>}
          <Button variant="outline" type="submit" width="full" mt={4}>
            {isLoading ? (
              <CircularProgress/>
            ) : (
              intl.formatMessage(messages.submit)
            )}
          </Button>
        </form>
      </Box>
    </Stack>
  );
}

export default Authorize;
