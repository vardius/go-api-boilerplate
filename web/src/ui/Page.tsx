import { useEffect, useId, useRef, type ReactNode } from 'react';
import { useIntl } from 'react-intl';
import styles from './Page.module.css';

interface PageProps {
  title: string;
  /** The browser tab's title, when it should differ from the heading. */
  documentTitle?: string;
  lead?: ReactNode;
  /**
   * Move focus to the heading when this page appears without a route change
   * (a form swapping to its result), so screen readers announce it.
   */
  focusOnMount?: boolean;
  children?: ReactNode;
}

/** A screen: its heading, the browser tab title, and its content. */
export function Page({ title, documentTitle, lead, focusOnMount = false, children }: PageProps) {
  const intl = useIntl();
  const heading = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    if (focusOnMount) heading.current?.focus();
  }, [focusOnMount]);

  return (
    <div className={styles.page}>
      <title>{intl.formatMessage({ id: 'app.documentTitle' }, { page: documentTitle ?? title })}</title>
      <header className={styles.header}>
        <h1 ref={heading} tabIndex={-1} className={styles.title}>
          {title}
        </h1>
        {lead !== undefined && <p className={styles.lead}>{lead}</p>}
      </header>
      {children}
    </div>
  );
}

interface SectionProps {
  title: string;
  /** Move focus to the heading when the section replaces what had focus, as Page does. */
  focusOnMount?: boolean;
  children: ReactNode;
}

/** A titled card within a page. */
export function Section({ title, focusOnMount = false, children }: SectionProps) {
  const id = useId();
  const heading = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    if (focusOnMount) heading.current?.focus();
  }, [focusOnMount]);

  return (
    <section className={styles.section} aria-labelledby={id}>
      <h2 id={id} ref={heading} tabIndex={focusOnMount ? -1 : undefined} className={styles.sectionTitle}>
        {title}
      </h2>
      {children}
    </section>
  );
}

/** An untitled card, for a page whose heading already names its one task. */
export function Card({ children }: { children: ReactNode }) {
  return <div className={styles.section}>{children}</div>;
}

export function Muted({ children }: { children: ReactNode }) {
  return <p className={styles.muted}>{children}</p>;
}

export function Stack({ children }: { children: ReactNode }) {
  return <div className={styles.stack}>{children}</div>;
}
