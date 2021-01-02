import React, {useCallback, useState} from "react";
import {defineMessages, useIntl} from "react-intl";
import {
  Box,
  Button,
  Checkbox,
  CheckboxGroup,
  CircularProgress,
  FormControl,
  FormLabel,
  HStack,
  Input
} from "@chakra-ui/core";
import {useApi} from "src/hooks";
import SubmitMessage from "src/components/common/SubmitMessage";

const messages = defineMessages({
  domain: {
    id: "create_client.form.domain",
    defaultMessage: "Domain",
  },
  redirect_url: {
    id: "create_client.form.redirect_url",
    defaultMessage: "Redirect URL",
  },
  scopes: {
    id: "create_client.form.scopes",
    defaultMessage: "Scopes",
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
    id: "create_client.form.submit",
    defaultMessage: "Create new client",
  },
  error: {
    id: "login.form.error",
    defaultMessage: "Action failed: {error}",
  },
});

export interface Props {
  onSuccess?: () => void;
}

const CreateClientForm = (props: Props) => {
  const intl = useIntl();
  const fetchJSON = useApi("auth");

  const [domain, setDomain] = useState("");
  const [redirectURL, setRedirectURL] = useState("");
  const [scopes, setScopes] = useState<Array<string>>([]);
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const createClient = useCallback(
    async (data: object) => {
      const body = JSON.stringify(data);

      return await fetchJSON(
        "/dispatch/client/client-create-credentials",
        "POST",
        null,
        body
      );
    },
    [fetchJSON]
  );

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    setIsLoading(true);

    if (domain && redirectURL && scopes.length > 0) {
      try {
        await createClient({domain, redirect_url: redirectURL, scopes});
        if (props.onSuccess) {
          props.onSuccess();
        }
      } catch (err) {
        setError(intl.formatMessage(messages.error, {error: err.message}));
      }
    }

    setDomain("");
    setRedirectURL("");
    setScopes([]);
    setIsLoading(false);
  };

  const handleSetScopes = (newScopes: []) => {
    setScopes(newScopes);
  };

  return (
    <Box
      p={8}
      maxWidth="500px"
      borderWidth={1}
      borderRadius={8}
      boxShadow="lg"
    >
      <Box my={4} textAlign="left">
        <form onSubmit={handleSubmit}>
          {error && <SubmitMessage message={error} status="error"/>}
          <FormControl isRequired>
            <FormLabel>{intl.formatMessage(messages.domain)}</FormLabel>
            <Input
              type="domain"
              size="lg"
              onChange={(event: React.ChangeEvent<HTMLInputElement>) =>
                setDomain(event.currentTarget.value)
              }
            />
          </FormControl>
          <FormControl isRequired>
            <FormLabel>{intl.formatMessage(messages.redirect_url)}</FormLabel>
            <Input
              type="redirect_url"
              size="lg"
              onChange={(event: React.ChangeEvent<HTMLInputElement>) =>
                setRedirectURL(event.currentTarget.value)
              }
            />
          </FormControl>
          <FormControl isRequired>
            <FormLabel>{intl.formatMessage(messages.scopes)}</FormLabel>
            <CheckboxGroup onChange={handleSetScopes} colorScheme="green" defaultValue={scopes}>
              <HStack>
                <Checkbox value="user_read">
                  {intl.formatMessage(messages.scopes_user_read)}
                </Checkbox>
                <Checkbox value="user_write">
                  {intl.formatMessage(messages.scopes_user_write)}
                </Checkbox>
              </HStack>
            </CheckboxGroup>
          </FormControl>
          <Button variant="outline" type="submit" width="full" mt={4}>
            {isLoading ? (
              <CircularProgress/>
            ) : (
              intl.formatMessage(messages.submit)
            )}
          </Button>
        </form>
      </Box>
    </Box>
  );
};

export default CreateClientForm;
