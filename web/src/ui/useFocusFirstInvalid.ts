import { useEffect, useRef, useState } from 'react';

/**
 * After a submit, moves focus to the form's first field marked invalid, so the
 * reader lands on what to fix. It runs again when the server's answer arrives,
 * so field errors from the server are covered too.
 *
 * Put `form` on the <form> and call `submitted()` from its onSubmit.
 */
export function useFocusFirstInvalid(serverError: unknown) {
  const form = useRef<HTMLFormElement>(null);
  const [attempts, setAttempts] = useState(0);

  useEffect(() => {
    if (attempts === 0) return;
    form.current?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus();
  }, [attempts, serverError]);

  return { form, submitted: () => setAttempts((count) => count + 1) };
}
