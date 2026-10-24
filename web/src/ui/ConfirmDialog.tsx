import { useEffect, useId, useRef, type ReactNode } from 'react';
import { useIntl } from 'react-intl';
import { Actions, Button } from './Button';
import styles from './ConfirmDialog.module.css';

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  confirmLabel: string;
  pendingLabel: string;
  pending: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  children: ReactNode;
}

/**
 * A modal question with Cancel and a destructive confirm. The native dialog
 * traps focus and closes on Escape; focus starts on Cancel and returns to the
 * control that opened it.
 */
export function ConfirmDialog({
  open,
  title,
  confirmLabel,
  pendingLabel,
  pending,
  onConfirm,
  onCancel,
  children,
}: ConfirmDialogProps) {
  const intl = useIntl();
  const dialog = useRef<HTMLDialogElement>(null);
  const cancel = useRef<HTMLButtonElement>(null);
  const opener = useRef<HTMLElement>(null);
  const titleId = useId();

  useEffect(() => {
    const element = dialog.current;
    if (!element) return;
    if (open && !element.open) {
      opener.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
      element.showModal();
      // showModal focuses the first control, the destructive one. Start on the safe choice instead.
      // (React's autoFocus runs at mount, while the dialog is still closed, so it can't do this.)
      cancel.current?.focus();
    }
    if (!open && element.open) {
      element.close();
      // The content unmounted before close(), so the browser had nothing focused to restore from.
      if (document.activeElement === null || document.activeElement === document.body) opener.current?.focus();
      opener.current = null;
    }
  }, [open]);

  return (
    <dialog
      ref={dialog}
      className={styles.dialog}
      aria-labelledby={titleId}
      onCancel={(event) => {
        // Escape asks the owner to close; the `open` prop stays the one source of truth.
        event.preventDefault();
        if (!pending) onCancel();
      }}
    >
      {open && (
        <div className={styles.content}>
          <h2 id={titleId} className={styles.title}>
            {title}
          </h2>
          {children}
          <Actions>
            <Button variant="danger" onClick={onConfirm} disabled={pending}>
              {pending ? pendingLabel : confirmLabel}
            </Button>
            <Button ref={cancel} onClick={onCancel} disabled={pending}>
              {intl.formatMessage({ id: 'common.cancel' })}
            </Button>
          </Actions>
        </div>
      )}
    </dialog>
  );
}
