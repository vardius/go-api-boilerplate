import React, {useCallback, useEffect, useState} from "react";
import {defineMessages, useIntl} from "react-intl";
import {
  Box,
  Button,
  ButtonGroup,
  CircularProgress,
  Code,
  Collapse,
  Flex,
  Heading,
  HStack,
  Link,
  Text,
  useToast
} from "@chakra-ui/core";
import AuthToken from "../AuthToken";
import {Token} from "src/types";
import {useApi} from "src/hooks";
import {API_URL} from "../../../../constants";

const clientCredentialsSnippet = `
package main

import (
  "context"
  "fmt"
	"log"

  "golang.org/x/oauth2/clientcredentials"
)

func main() {
  cfg := clientcredentials.Config{
    ClientID:     "__CLIENT_ID__",
    ClientSecret: "__CLIENT_SECRET__",
    TokenURL:     "__API_URL__/auth/v1/token",
  }

  token, err := cfg.Token(context.Background())
  if err != nil {
    log.Fatal(err)
  }

  fmt.Println(token)
}
`;

const authCodeSnippet = `
package main

import (
  "context"
  "encoding/json"
  "log"
  "net/http"

  "golang.org/x/oauth2"
)

func main() {
  config := oauth2.Config{
    ClientID:     "__CLIENT_ID__",
    ClientSecret: "__CLIENT_SECRET__",
    RedirectURL:  "__CLIENT_REDIRECT_URL__",
		Scopes:       []string{"user_read", "user_write"},
    Endpoint: oauth2.Endpoint{
      AuthURL:  "__API_URL__/auth/v1/authorize",
      TokenURL: "__API_URL__/auth/v1/token",
    },
  }

  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    u := config.AuthCodeURL("xyz")
    http.Redirect(w, r, u, http.StatusFound)
  })

  // your __CLIENT_REDIRECT_URL__
  http.HandleFunc("/oauth2", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse request", http.StatusBadRequest)
			return
		}
    state := r.Form.Get("state")
    if state != "xyz" {
      http.Error(w, "State invalid", http.StatusBadRequest)
      return
    }
    code := r.Form.Get("code")
    if code == "" {
      http.Error(w, "Code not found", http.StatusBadRequest)
      return
    }
    token, err := config.Exchange(r.Context(), code)
    if err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
    }

    e := json.NewEncoder(w)
    e.SetIndent("", "  ")
    e.Encode(token)
  })

  log.Println("Client is running at 3000 port.Please open http://localhost:3000")
  log.Fatal(http.ListenAndServe(":3000", nil))
}
`;

const tokenPage = 1
const tokenLimit = 999

const messages = defineMessages({
  show_tokens: {
    id: "security.client.show",
    defaultMessage: "Show tokens",
  },
  hide_tokens: {
    id: "security.client.hide_tokens",
    defaultMessage: "Hide tokens",
  },
  show_snippet: {
    id: "security.client.show_snippet",
    defaultMessage: "Show code snippet",
  },
  hide_snippet: {
    id: "security.client.hide_snippet",
    defaultMessage: "Hide code snippet",
  },
  remove: {
    id: "security.client.remove",
    defaultMessage: "Delete credentials",
  },
  clientID: {
    id: "security.client_credentials.client_id",
    defaultMessage: "Client ID: {value}",
  },
  clientSecret: {
    id: "security.client_credentials.client_secret",
    defaultMessage: "Secret: {value}",
  },
  scopes: {
    id: "security.client_credentials.client_scopes",
    defaultMessage: "Scopes: {value}",
  },
  client_credentials_snippet: {
    id: "security.client_credentials.client_credentials_snippet",
    defaultMessage: "Client credentials snippet",
  },
  auth_code_snippet: {
    id: "security.client_credentials.auth_code_snippet",
    defaultMessage: "Auth code snippet",
  },
  remove_error: {
    id: "security.client_credentials.remove.error",
    defaultMessage: "{error}",
  },
  remove_error_title: {
    id: "security.client_credentials.remove.error_title",
    defaultMessage: "Action failed",
  },
});

interface Props {
  domain: string;
  redirectURL: string;
  clientID: string;
  clientSecret: string;
  scopes: Array<string>;
  onRemove?: () => void;
}

const ClientCredentials = (props: Props) => {
  const intl = useIntl();
  const toast = useToast();
  const fetchJSON = useApi("auth");

  const [isLoading, setIsLoading] = useState(false);
  const [isTokensLoading, setIsTokensLoading] = useState(false);
  const [tokens, setTokens] = useState<Array<Token>>([]);
  const [showTokens, setShowTokens] = React.useState(false)
  const [showSnippet, setShowSnippet] = React.useState(false)

  const clientID = props.clientID;
  const onRemove = props.onRemove;

  const fetchAuthTokens = useCallback(
    async ({page, limit}: { page: number; limit: number }) => {
      return await fetchJSON(`/clients/${clientID}/tokens`, "GET", new URLSearchParams({
        page: String(page),
        limit: String(limit),
      }));
    },
    [fetchJSON, clientID]
  );

  const remove = useCallback(
    async (data: object) => {
      const body = JSON.stringify(data);

      await fetchJSON(
        "/dispatch/client/client-remove-credentials",
        "POST",
        null,
        body
      );

      if (onRemove) {
        onRemove();
      }
    },
    [fetchJSON, onRemove]
  );

  useEffect(() => {
    let mounted = true
    if (!showTokens) {
      setIsTokensLoading(false);
      return;
    }

    const load = async () => {
      try {
        const response = await fetchAuthTokens({page: tokenPage, limit: tokenLimit});
        if (!mounted) {
          return
        }

        setIsTokensLoading(false);
        setTokens(response.auth_tokens || []);
      } catch (err) {
        if (!mounted) {
          return
        }

        setIsTokensLoading(false);
      }
    };

    load();

    return function cleanup() {
      mounted = false
    }
  }, [showTokens, fetchAuthTokens]);

  const handleToggleTokens = () => setShowTokens(!showTokens)
  const handleToggleSnippet = () => setShowSnippet(!showSnippet)

  const handleRemove = async () => {
    setIsLoading(true);

    try {
      await remove({id: props.clientID});
    } catch (err) {
      toast({
        position: "top",
        title: intl.formatMessage(messages.remove_error_title),
        description: intl.formatMessage(messages.remove_error, {error: err.message}),
        status: "error",
        duration: 9000,
        isClosable: true,
      });
    }

    setIsLoading(false);
  };

  return (
    <Box>
      <Flex justify="space-between">
        <Box mt={{base: 4, md: 0}} ml={{md: 6}}>
          <Text
            fontWeight="bold"
            textTransform="uppercase"
            fontSize="sm"
            letterSpacing="wide"
            color="teal.600"
          >
            {props.domain}
          </Text>
          <Link
            mt={1}
            display="block"
            fontSize="lg"
            lineHeight="normal"
            fontWeight="semibold"
            href={props.redirectURL}
          >
            {props.redirectURL}
          </Link>
          <Text mt={2} color="gray.500">
            {intl.formatMessage(messages.clientID, {value: props.clientID})}
          </Text>
          <Text mt={2} color="gray.500">
            {intl.formatMessage(messages.clientSecret, {value: props.clientSecret})}
          </Text>
          <Text mt={2} color="gray.500">
            {intl.formatMessage(messages.scopes, {value: props.scopes.join(',')})}
          </Text>
          <ButtonGroup variant="outline" spacing="6">
            <Button size="sm" onClick={handleToggleTokens} mt="1rem">
              {showTokens ? intl.formatMessage(messages.hide_tokens) : intl.formatMessage(messages.show_tokens)}
            </Button>
            <Button size="sm" onClick={handleToggleSnippet} mt="1rem">
              {showSnippet ? intl.formatMessage(messages.hide_snippet) : intl.formatMessage(messages.show_snippet)}
            </Button>
          </ButtonGroup>
        </Box>
        <Button
          isLoading={isLoading}
          colorScheme="red"
          variant="outline"
          onClick={handleRemove}
        >
          {intl.formatMessage(messages.remove)}
        </Button>
      </Flex>
      <Collapse isOpen={showTokens}>
        {isTokensLoading ? (
          <CircularProgress/>
        ) : (
          tokens.map(({id, access}) => <AuthToken key={id} id={id} authToken={access} title={id}/>)
        )}
      </Collapse>
      <Collapse isOpen={showSnippet}>
        <Box p={5} m={2} shadow="md" borderWidth="1px">
          <HStack justifyContent="space-around" alignItems="flex-start">
            <Box>
              <Heading fontSize="xl">{intl.formatMessage(messages.client_credentials_snippet)}</Heading>
              <Code>
                <pre>
                  {
                    clientCredentialsSnippet
                      .split('__CLIENT_ID__').join(props.clientID)
                      .split('__CLIENT_SECRET__').join(props.clientSecret)
                      .split('__CLIENT_REDIRECT_URL__').join(props.redirectURL)
                      .split('__API_URL__').join(API_URL)
                  }
                </pre>
              </Code>
            </Box>
            <Box>
              <Heading fontSize="xl">{intl.formatMessage(messages.auth_code_snippet)}</Heading>
              <Code>
                <pre>
                  {
                    authCodeSnippet
                      .split('__CLIENT_ID__').join(props.clientID)
                      .split('__CLIENT_SECRET__').join(props.clientSecret)
                      .split('__CLIENT_REDIRECT_URL__').join(props.redirectURL)
                      .split('__API_URL__').join(API_URL)
                  }
                </pre>
              </Code>
            </Box>
          </HStack>
        </Box>
      </Collapse>
    </Box>
  );
};

export default ClientCredentials;
