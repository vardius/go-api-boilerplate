import { useContext } from "react";
import { UserContext } from "src/context/UserContext";

export default function useLocale() {
  return useContext(UserContext);
}
