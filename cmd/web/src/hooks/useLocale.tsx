import {useContext} from "react";
import {LocaleContext} from "src/context/LocaleContext";

export default function useLocale() {
  return useContext(LocaleContext);
}
