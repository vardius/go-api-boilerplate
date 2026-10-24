import { useId, type ComponentProps, type ReactNode } from 'react';
import styles from './Field.module.css';

interface FieldProps {
  label: string;
  hint?: string | undefined;
  error?: string | undefined;
}

function useFieldIds({ hint, error }: FieldProps) {
  const id = useId();
  const hintId = `${id}-hint`;
  const errorId = `${id}-error`;
  const describedBy = [hint === undefined ? '' : hintId, error === undefined ? '' : errorId].join(' ').trim();
  return {
    id,
    hintId,
    errorId,
    control: {
      id,
      'aria-describedby': describedBy === '' ? undefined : describedBy,
      'aria-invalid': error === undefined ? undefined : true,
    },
  };
}

function FieldFrame({
  label,
  hint,
  error,
  ids,
  children,
}: FieldProps & { ids: ReturnType<typeof useFieldIds>; children: ReactNode }) {
  return (
    <div className={styles.field}>
      <label htmlFor={ids.id} className={styles.label}>
        {label}
      </label>
      {hint !== undefined && (
        <p id={ids.hintId} className={styles.hint}>
          {hint}
        </p>
      )}
      {children}
      {error !== undefined && (
        <p id={ids.errorId} className={styles.error}>
          {error}
        </p>
      )}
    </div>
  );
}

type InputProps = Omit<ComponentProps<'input'>, 'id' | 'className' | 'aria-describedby' | 'aria-invalid'>;

export function TextField({ label, hint, error, ...input }: FieldProps & InputProps) {
  const ids = useFieldIds({ label, hint, error });
  return (
    <FieldFrame label={label} hint={hint} error={error} ids={ids}>
      <input className={styles.control} {...ids.control} {...input} />
    </FieldFrame>
  );
}

type TextAreaProps = Omit<ComponentProps<'textarea'>, 'id' | 'className' | 'aria-describedby' | 'aria-invalid'>;

export function TextAreaField({ label, hint, error, ...textarea }: FieldProps & TextAreaProps) {
  const ids = useFieldIds({ label, hint, error });
  return (
    <FieldFrame label={label} hint={hint} error={error} ids={ids}>
      <textarea className={styles.control} {...ids.control} {...textarea} />
    </FieldFrame>
  );
}

interface CheckboxGroupProps {
  legend: string;
  hint?: string | undefined;
  error?: string | undefined;
  children: ReactNode;
}

export function CheckboxGroup({ legend, hint, error, children }: CheckboxGroupProps) {
  const id = useId();
  const describedBy = [hint === undefined ? '' : `${id}-hint`, error === undefined ? '' : `${id}-error`]
    .join(' ')
    .trim();
  return (
    <fieldset className={styles.fieldset} aria-describedby={describedBy === '' ? undefined : describedBy}>
      <legend className={styles.legend}>{legend}</legend>
      {hint !== undefined && (
        <p id={`${id}-hint`} className={styles.hint}>
          {hint}
        </p>
      )}
      {children}
      {error !== undefined && (
        <p id={`${id}-error`} className={styles.error}>
          {error}
        </p>
      )}
    </fieldset>
  );
}

interface CheckboxProps extends Omit<ComponentProps<'input'>, 'type' | 'className'> {
  name: string;
  label: string;
  description: string;
}

export function Checkbox({ label, description, ...input }: CheckboxProps) {
  const id = useId();
  return (
    <label className={styles.check}>
      <input type="checkbox" aria-labelledby={`${id}-name`} aria-describedby={`${id}-description`} {...input} />
      <span className={styles.checkText}>
        <span id={`${id}-name`} className={styles.checkName}>
          {label}
        </span>
        <span id={`${id}-description`} className={styles.checkDescription}>
          {description}
        </span>
      </span>
    </label>
  );
}
