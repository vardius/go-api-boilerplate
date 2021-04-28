import React, {createContext, ReactChild, useCallback, useEffect,} from "react";
import {useCookies} from "react-cookie";
import {useHistory} from "react-router-dom";
import {AUTH_TOKEN_COOKIE} from "src/constants";
import {useQuery} from "src/hooks";
import {AuthToken} from "src/types";

const cookieOptions = {
  domain: window.location.hostname,
  path: "/",
  maxAge: 60 * 60 * 24 * 365, // 365 days in seconds
};

export const AuthContext = createContext<[AuthToken, () => void]>([
  null,
  () => {
  },
]);

export interface Props {
  children: ReactChild;
}

const AuthContextProvider = (props: Props) => {
  const query = useQuery();
  const history = useHistory();
  const [cookies, setCookie, removeCookie] = useCookies([AUTH_TOKEN_COOKIE]);

  const authToken = query.get("authToken");

  const removeAuthToken = useCallback(() => {
    removeCookie(AUTH_TOKEN_COOKIE, cookieOptions);
  }, [removeCookie]);

  const setAuthToken = useCallback(
    (token: AuthToken) => {
      if (!token || token === "none") {
        removeAuthToken();
      } else if (token && token.length > 0) {
        setCookie(AUTH_TOKEN_COOKIE, authToken, cookieOptions);
      }

      query.delete("authToken");

      history.replace({
        search: query.toString()
      });
    },
    [authToken, removeAuthToken, setCookie, history, query]
  );

  useEffect(() => {
    if (authToken) {
      setAuthToken(authToken);
    }
  }, [authToken, setAuthToken]);

  return (
    <AuthContext.Provider
      value={[cookies[AUTH_TOKEN_COOKIE] || null, () => removeAuthToken()]}
    >
      {props.children}
    </AuthContext.Provider>
  );
};

export default AuthContextProvider;
