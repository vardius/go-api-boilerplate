import React, {useCallback, useEffect, useState} from "react";
import {defineMessages, useIntl} from "react-intl";
import {Box, Button, Center, Checkbox, CheckboxGroup, CircularProgress, Heading, HStack, Stack} from "@chakra-ui/core";
import {useApi, useQuery, useUser} from "src/hooks";
import {SubmitMessage} from "src/components/common";

type Client = { domain: string } | null;

const messages = defineMessages({
  title: {
    id: "authorize.title",
    defaultMessage: "Authorize",
  },
  scopes_user_read: {
    id: "create_client.form.scopes_user_read",
    defaultMessage: "User read",
  },
  scopes_user_write: {
    id: "create_client.form.scopes_user_write",
    defaultMessage: "User write",
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
  const fetchJSON = useApi("auth");
  const query = useQuery();
  const [user] = useUser();
  const [error, setError] = useState("");
  const [client, setClient] = React.useState(null as Client);
  const [isFetching, setIsFetching] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const clientID = query.get('client_id');
  const scope = query.get('scope');

  const fetchClient = useCallback(async (): Promise<Client> => {
    const json = await fetchJSON(`/clients/${clientID}`, "GET");

    return json as Client;
  }, [fetchJSON, clientID]);

  const authorize = useCallback(
    async () => {
      if (!user) {
        return null;
      }

      const response = await fetchJSON(`/authorize`, "POST", query);

      window.location.assign(response.location);
    },
    [fetchJSON, user, query]
  );

  useEffect(() => {
    let mounted = true
    const load = async () => {
      try {
        const response = await fetchClient();
        if (!mounted) {
          return
        }

        setClient(response);
      } catch (err) {
        if (!mounted) {
          return
        }

        setClient(null);
      }

      setIsFetching(false);
    };

    if (clientID) {
      load();
    }

    return function cleanup() {
      mounted = false
    }
  }, [clientID, fetchClient]);

  if (!user || isFetching) {
    return null;
  }

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    setIsSubmitting(true);

    try {
      await authorize();
    } catch (err) {
      setError(intl.formatMessage(messages.error, {error: err.message}));
    }

    setIsSubmitting(false);
  };

  return (
    <Center minHeight="100vh">
      <Box
        p={8}
        maxWidth="500px"
        borderWidth={1}
        borderRadius={8}
        boxShadow="lg"
      >
        <Center>
          <Stack>
            <Heading>{intl.formatMessage(messages.title)}</Heading>
            {client && <Heading>{client.domain}</Heading>}
          </Stack>
        </Center>
        <Box my={4} textAlign="left">
          <form onSubmit={handleSubmit}>
            {error && <SubmitMessage message={error} status="error"/>}
            {scope && <CheckboxGroup colorScheme="green" defaultValue={scope.split(' ')}>
              <HStack>
                <Checkbox isDisabled isReadOnly value="user_read">
                  {intl.formatMessage(messages.scopes_user_read)}
                </Checkbox>
                <Checkbox isDisabled isReadOnly value="user_write">
                  {intl.formatMessage(messages.scopes_user_write)}
                </Checkbox>
              </HStack>
            </CheckboxGroup>}
            <Button variant="outline" type="submit" width="full" mt={4}>
              {isSubmitting ? (
                <CircularProgress/>
              ) : (
                intl.formatMessage(messages.submit)
              )}
            </Button>
          </form>
        </Box>
      </Box>
    </Center>
  );
}

export default Authorize;
