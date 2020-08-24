import {useMemo} from "react";
import {useAuthToken} from "src/hooks";
import {fetchJSON} from "src/api";

export default function useApi() {
  const [authToken] = useAuthToken();

  return useMemo(() => fetchJSON(authToken), [authToken]);
}
