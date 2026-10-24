import { useEffect, useRef, type ReactNode } from 'react';
import styles from './Notice.module.css';

type Tone = 'info' | 'success' | 'warning' | 'danger';

interface NoticeProps {
  tone?: Tone;
  title?: string;
  /**
   * Announce the notice when it appears: "alert" interrupts (errors), "status"
   * waits its turn (confirmations). Leave unset for notices present on load.
   */
  live?: 'alert' | 'status';
  /**
   * Take focus when it appears, for a notice that replaces the control that had
   * focus (a form swapping to its result): screen readers then read it out.
   */
  focusOnMount?: boolean;
  children?: ReactNode;
}

export function Notice({ tone = 'info', title, live, focusOnMount = false, children }: NoticeProps) {
  const notice = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (focusOnMount) notice.current?.focus();
  }, [focusOnMount]);

  return (
    <div
      ref={notice}
      tabIndex={focusOnMount ? -1 : undefined}
      className={`${styles.notice ?? ''} ${styles[tone] ?? ''}`}
      role={live}
    >
      {title !== undefined && <p className={styles.title}>{title}</p>}
      {children !== undefined && <div className={styles.body}>{children}</div>}
    </div>
  );
}
