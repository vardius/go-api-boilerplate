import {useContext} from "react";
import {AuthContext} from "src/context/AuthContext";

export default function useAuthToken() {
  return useContext(AuthContext);
}
