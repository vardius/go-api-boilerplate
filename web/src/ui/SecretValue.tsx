import { useId, useState } from 'react';
import { useIntl } from 'react-intl';
import { Button } from './Button';
import styles from './SecretValue.module.css';

interface SecretValueProps {
  label: string;
  /** The copy button's accessible name, e.g. "Copy client secret". */
  copyLabel: string;
  value: string;
}

/** A value to copy exactly: shown in full, selectable in one click, with a copy button. */
export function SecretValue({ label, copyLabel, value }: SecretValueProps) {
  const intl = useIntl();
  const id = useId();
  const [copy, setCopy] = useState<'idle' | 'copied' | 'failed'>('idle');

  async function copyValue() {
    try {
      await navigator.clipboard.writeText(value);
      setCopy('copied');
    } catch {
      // No clipboard (an insecure origin, or permission denied): the value stays selectable.
      setCopy('failed');
    }
  }

  return (
    <div className={styles.secret} role="group" aria-labelledby={`${id}-label`}>
      <span id={`${id}-label`} className={styles.label}>
        {label}
      </span>
      <div className={styles.row}>
        <code className={styles.value}>{value}</code>
        <Button aria-label={copyLabel} onClick={() => void copyValue()}>
          {intl.formatMessage({ id: copy === 'copied' ? 'copy.copied' : 'copy.copy' })}
        </Button>
      </div>
      <p className={copy === 'failed' ? styles.status : 'visually-hidden'} role="status">
        {copy === 'idle' ? '' : intl.formatMessage({ id: copy === 'copied' ? 'copy.copied' : 'copy.failed' })}
      </p>
    </div>
  );
}
