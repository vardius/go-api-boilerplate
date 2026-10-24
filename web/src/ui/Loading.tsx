import { useIntl } from 'react-intl';
import styles from './Loading.module.css';

export function Loading({ label }: { label?: string }) {
  const intl = useIntl();
  return (
    <p className={styles.loading} role="status">
      <span className={styles.spinner} aria-hidden="true" />
      {label ?? intl.formatMessage({ id: 'common.loading' })}
    </p>
  );
}
