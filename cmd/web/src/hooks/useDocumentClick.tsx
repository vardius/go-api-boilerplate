import {useEffect} from "react";

export default function useDocumentClick(
  handler: EventListenerOrEventListenerObject
) {
  useEffect(() => {
    document.addEventListener("click", handler);

    return () => {
      document.removeEventListener("click", handler);
    };
  }, [handler]);

  return null;
}
