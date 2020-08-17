import React, {
  createContext,
  ReactChild,
  useEffect,
  useCallback,
} from "react";
import { useCookies } from "react-cookie";
import { useLocation } from "react-router-dom";
import { AUTH_TOKEN_COOKIE } from "src/constants";
import { AuthToken } from "src/types";

function useQuery() {
  return new URLSearchParams(useLocation().search);
}

export const AuthContext = createContext<[AuthToken, () => void]>([
  null,
  () => {},
]);

export interface Props {
  children: ReactChild;
}

const AuthContextProvider = (props: Props) => {
  const query = useQuery();
  const [cookies, setCookie, removeCookie] = useCookies([AUTH_TOKEN_COOKIE]);

  const authToken = query.get("authToken");

  const removeAuthToken = useCallback(() => {
    removeCookie(AUTH_TOKEN_COOKIE)
  }, [removeCookie]);

  const setAuthToken = useCallback(
    (token: AuthToken) => {
      if (token === "none") {
        removeAuthToken();
      } else if (token && token.length > 0) {
        const cookieOptions = {
          domain: window.location.hostname,
          path: "/",
          maxAge: 60 * 60 * 24 * 365, // 365 days in seconds
        };

        setCookie(AUTH_TOKEN_COOKIE, authToken, cookieOptions);
      }
    },
    [authToken, removeAuthToken, setCookie]
  );

  useEffect(() => {
    setAuthToken(authToken);
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
