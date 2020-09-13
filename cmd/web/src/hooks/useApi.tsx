import {useMemo} from "react";
import {useAuthToken} from "src/hooks";
import {fetchJSON} from "src/api";
import {API_AUTH_URL, API_URL, API_USERS_URL} from "src/constants";

type API = "users" | "auth" | null

export default function useApi(api?: API) {
  const [authToken] = useAuthToken();

  let baseURL = API_URL
  switch (api) {
    case "users": {
      baseURL = API_USERS_URL
      break;
    }
    case "auth": {
      baseURL = API_AUTH_URL
      break;
    }
  }

  return useMemo(() => fetchJSON(baseURL, authToken), [baseURL, authToken]);
}
