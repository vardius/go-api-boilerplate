import {useContext} from "react";
import {MessagesContext} from "./components/MessagesContext";

export default function useMessages() {
  return useContext(MessagesContext);
}
