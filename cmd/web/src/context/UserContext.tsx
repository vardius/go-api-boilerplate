import React, {
  createContext,
  ReactChild,
  useEffect,
  useCallback,
} from "react";
import { User } from "src/types";
import { UnauthorizedHttpError } from "src/errors";
import { useAuthToken, useApi } from "src/hooks";

type user = User | null;

export const UserContext = createContext<
  [user, React.Dispatch<React.SetStateAction<user>>]
>([null, () => {}]);

export interface Props {
  children: ReactChild;
}

const UserContextProvider = (props: Props) => {
  const [user, setUser] = React.useState(null as user);
  const [authToken] = useAuthToken();
  const fetchJSON = useApi();

  const fetchMe = useCallback(async (): Promise<user> => {
    try {
      const json = await fetchJSON(`/users/v1/me`, "GET");

      return json as User;
    } catch (err) {
      if (err instanceof UnauthorizedHttpError) {
        return null;
      }

      throw err;
    }
  }, [fetchJSON]);

  useEffect(() => {
    const load = async () => {
      try {
        const response = await fetchMe();

        setUser(response);
      } catch (err) {
        console.error(err);
        setUser(null);
      }

      
    };

    if (authToken) {
      load();
    }
  }, [authToken, fetchMe]);

  return (
    <UserContext.Provider value={[user, setUser]}>
      {props.children}
    </UserContext.Provider>
  );
};

export default UserContextProvider;
