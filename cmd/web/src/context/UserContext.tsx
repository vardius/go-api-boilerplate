import React, { createContext, ReactChild, useEffect } from "react";
import { User, AuthToken } from "src/types";
import { UnauthorizedHttpError, fetchJSON } from "src/api";
import { useAuthToken } from "src/hooks";

type user = User | null;

const fetchMe = async (authToken?: AuthToken): Promise<user> => {
  if (!authToken) {
    return null;
  }
  try {
    const json = await fetchJSON(`/users/v1/me?authToken=${authToken}`, "GET");

    return json as User;
  } catch (err) {
    if (err instanceof UnauthorizedHttpError) {
      throw err;
    }
  }

  return null;
};

export const UserContext = createContext<
  [user, React.Dispatch<React.SetStateAction<user>>]
>([null, () => {}]);

export interface Props {
  children: ReactChild;
}

const UserContextProvider = (props: Props) => {
  const [authToken] = useAuthToken();
  const [user, setUser] = React.useState(null as user);

  useEffect(() => {
    const load = async () => {
      try {
        const response = await fetchMe(authToken);

        setUser(response);
      } catch (err) {
        console.error(err);
        setUser(null);
      }
    };

    load();
  }, [authToken]);

  return (
    <UserContext.Provider value={[user, setUser]}>
      {props.children}
    </UserContext.Provider>
  );
};

export default UserContextProvider;
