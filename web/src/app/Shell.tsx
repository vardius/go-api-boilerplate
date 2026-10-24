import { useEffect, useId, useRef } from 'react';
import { useIntl } from 'react-intl';
import { Link, NavLink, Outlet, useLocation } from 'react-router';
import { useLocale } from '../i18n/locale';
import { isLocale } from '../i18n/messages';
import styles from './Shell.module.css';

/** Moves focus to the new page's heading after a client-side navigation, as a full page load would. */
function useFocusHeadingOnNavigate() {
  const { pathname } = useLocation();
  const previous = useRef(pathname);
  useEffect(() => {
    if (previous.current === pathname) return;
    previous.current = pathname;
    document.querySelector<HTMLElement>('main h1')?.focus();
  }, [pathname]);
}

function LanguageSelect() {
  const intl = useIntl();
  const { locale, setLocale } = useLocale();
  const id = useId();
  return (
    <>
      <label htmlFor={id} className={styles.footerLabel}>
        {intl.formatMessage({ id: 'footer.language' })}
      </label>
      <select
        id={id}
        className={styles.select}
        value={locale}
        onChange={(event) => {
          if (isLocale(event.target.value)) setLocale(event.target.value);
        }}
      >
        <option value="en" lang="en">
          English
        </option>
        <option value="pl" lang="pl">
          Polski
        </option>
      </select>
    </>
  );
}

function AccountNav() {
  const intl = useIntl();
  const links = [
    { to: '/profile', label: intl.formatMessage({ id: 'nav.profile' }) },
    { to: '/devices', label: intl.formatMessage({ id: 'nav.devices' }) },
    { to: '/apps', label: intl.formatMessage({ id: 'nav.apps' }) },
  ];
  return (
    <nav aria-label={intl.formatMessage({ id: 'nav.label' })}>
      <ul className={styles.navList}>
        {links.map((link) => (
          <li key={link.to}>
            <NavLink to={link.to} className={styles.navLink ?? ''}>
              {link.label}
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  );
}

function Shell({ nav }: { nav: boolean }) {
  const intl = useIntl();
  return (
    <div className={styles.shell}>
      <a href="#main" className={styles.skip}>
        {intl.formatMessage({ id: 'app.skipToContent' })}
      </a>
      <header className={styles.header}>
        <div className={styles.headerInner}>
          <Link to="/" className={styles.brand}>
            {intl.formatMessage({ id: 'app.name' })}
          </Link>
          {nav && <AccountNav />}
        </div>
      </header>
      <main id="main" tabIndex={-1} className={styles.main}>
        <Outlet />
      </main>
      <footer className={styles.footer}>
        <div className={styles.footerInner}>
          <LanguageSelect />
        </div>
      </footer>
    </div>
  );
}

/** The top route: everything below it is a screen. */
export function Root() {
  useFocusHeadingOnNavigate();
  return <Outlet />;
}

/** Screens behind sign-in, with the account navigation. */
export function AccountLayout() {
  return <Shell nav />;
}

/** Screens anyone can reach: sign-in, consent, errors. */
export function PublicLayout() {
  return <Shell nav={false} />;
}
