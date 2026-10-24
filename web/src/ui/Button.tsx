import type { ComponentProps, ReactNode } from 'react';
import { Link, type LinkProps } from 'react-router';
import styles from './Button.module.css';

export type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'dangerQuiet';

function buttonClass(variant: ButtonVariant): string {
  return `${styles.button ?? ''} ${styles[variant] ?? ''}`;
}

interface ButtonProps extends Omit<ComponentProps<'button'>, 'className'> {
  variant?: ButtonVariant;
}

export function Button({ variant = 'secondary', type = 'button', ...props }: ButtonProps) {
  return <button type={type} className={buttonClass(variant)} {...props} />;
}

interface ButtonLinkProps extends Omit<LinkProps, 'className'> {
  variant?: ButtonVariant;
}

/** A client-side route that looks like a button. */
export function ButtonLink({ variant = 'secondary', ...props }: ButtonLinkProps) {
  return <Link className={buttonClass(variant)} {...props} />;
}

interface ButtonAnchorProps extends Omit<ComponentProps<'a'>, 'className'> {
  variant?: ButtonVariant;
}

/** A full page load (a server route or another site) that looks like a button. */
export function ButtonAnchor({ variant = 'secondary', ...props }: ButtonAnchorProps) {
  return <a className={buttonClass(variant)} {...props} />;
}

/** Lays out a row of buttons that wraps on narrow screens. */
export function Actions({ children }: { children: ReactNode }) {
  return <div className={styles.actions}>{children}</div>;
}
