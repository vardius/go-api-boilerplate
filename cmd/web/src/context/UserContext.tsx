import React, {createContext, ReactChild, useCallback, useEffect, useState,} from "react";
import {User} from "src/types";
import {UnauthorizedHttpError} from "src/errors";
import {useApi, useAuthToken} from "src/hooks";
import {Center, CircularProgress} from "@chakra-ui/core";

type user = User | null;

export const UserContext = createContext<[user, React.Dispatch<React.SetStateAction<user>>]>([null, () => {
}]);

export interface Props {
  children: ReactChild;
}

const UserContextProvider = (props: Props) => {
  const [user, setUser] = React.useState(null as user);
  const [authToken, logout] = useAuthToken();
  const fetchJSON = useApi("users");
  const [isLoading, setIsLoading] = useState(!!authToken);

  const fetchMe = useCallback(async (): Promise<user> => {
    try {
      const json = await fetchJSON(`/me`, "GET");

      return json as User;
    } catch (err) {
      if (err instanceof UnauthorizedHttpError) {
        logout();

        return null;
      }

      throw err;
    }
  }, [fetchJSON, logout]);

  useEffect(() => {
    let mounted = true
    const load = async () => {
      try {
        const response = await fetchMe();
        if (!mounted) {
          return
        }

        setUser(response);
      } catch (err) {
        if (!mounted) {
          return
        }

        setUser(null);
      }

      setIsLoading(false);
    };

    if (authToken) {
      load();
    }

    return function cleanup() {
      mounted = false
    }
  }, [authToken, fetchMe]);

  return (
    <UserContext.Provider value={[user, setUser]}>
      {isLoading ? (
        <Center minHeight="100vh">
          <CircularProgress/>
        </Center>
      ) : (
        props.children
      )}
    </UserContext.Provider>
  );
};

export default UserContextProvider;
