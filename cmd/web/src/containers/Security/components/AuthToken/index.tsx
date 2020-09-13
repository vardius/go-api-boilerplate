import React, {useCallback, useState} from "react";
import {defineMessages, useIntl} from "react-intl";
import {Box, Button, Code, Flex, Heading, Text, useToast} from "@chakra-ui/core";
import {useApi} from "src/hooks";

interface Props {
  id: string;
  authToken: string;
  title: string;
  onRemove?: () => void;
}

const messages = defineMessages({
  remove: {
    id: "security.auth_token.remove",
    defaultMessage: "Invalidate session",
  },
  remove_error: {
    id: "security.auth_token.remove.error",
    defaultMessage: "{error}",
  },
  remove_error_title: {
    id: "security.auth_token.remove.error_title",
    defaultMessage: "Action failed",
  },
});

const AuthToken = (props: Props) => {
  const intl = useIntl();
  const toast = useToast();
  const fetchJSON = useApi("auth");

  const [isLoading, setIsLoading] = useState(false);

  const onRemove = props.onRemove;

  const remove = useCallback(
    async (data: object) => {
      const body = JSON.stringify(data);

      await fetchJSON(
        "/dispatch/token/remove-auth-token",
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

  const handleRemove = async () => {
    setIsLoading(true);

    try {
      await remove({id: props.id});
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
    <Box p={5} m={2} shadow="md" borderWidth="1px">
      <Flex justify="space-between">
        <Heading fontSize="xl">{props.title}</Heading>
        <Button
          isLoading={isLoading}
          colorScheme="red"
          variant="outline"
          onClick={handleRemove}
        >
          {intl.formatMessage(messages.remove)}
        </Button>
      </Flex>
      <pre>
        <Text mt={4}>
            <Code>
              {props.authToken}
            </Code>
        </Text>
      </pre>
    </Box>
  );
};

export default AuthToken;
