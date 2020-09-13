import React, {useCallback, useEffect, useState} from "react";
import {Box} from "@chakra-ui/core";
import {useApi, useUser} from "src/hooks";
import {Token} from "src/types";
import AuthToken from "../AuthToken";

const page = 1
const limit = 999

function AuthTokenList() {
  const fetchJSON = useApi("auth");

  const [user] = useUser();
  const [isLoading, setIsLoading] = useState(true);
  const [tokens, setTokens] = useState<Array<Token>>([]);

  const fetchAuthTokens = useCallback(
    async ({page, limit}: { page: number; limit: number }) => {
      if (!user) {
        return null;
      }

      return await fetchJSON(`/users/${user.id}/tokens`, "GET", new URLSearchParams({
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
  }, [user, fetchAuthTokens]);

  if (!user || isLoading || tokens.length === 0) {
    return null;
  }

  const handleRemoveToken = async () => {
    const response = await fetchAuthTokens({page, limit});
    setTokens(response.auth_tokens || []);
  };

  return (
    <Box>
      {tokens.map(({id, access, user_agent}) => <AuthToken
        key={id}
        id={id}
        authToken={access}
        title={user_agent || id}
        onRemove={handleRemoveToken}
      />)}
    </Box>
  );
}

export default AuthTokenList;
