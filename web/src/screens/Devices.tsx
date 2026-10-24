import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useId, useState } from 'react';
import { useIntl, type IntlShape } from 'react-intl';
import { useNavigate } from 'react-router';
import { isProblem } from '../api/describe';
import { deleteCurrentSession, revokeSession } from '../api/endpoints';
import { queryKeys, useSessions } from '../api/queries';
import type { Session } from '../api/schemas';
import type { SignInReason } from '../app/queryClient';
import { describeUserAgent } from '../lib/device';
import { Actions, Button } from '../ui/Button';
import { ErrorState } from '../ui/ErrorState';
import { Loading } from '../ui/Loading';
import { Notice } from '../ui/Notice';
import { Page, Section } from '../ui/Page';
import styles from './screens.module.css';

function deviceName(intl: IntlShape, session: Session): string {
  const { browser, os } = describeUserAgent(session.user_agent);
  const name = browser ?? intl.formatMessage({ id: 'devices.unknownBrowser' });
  return os === undefined ? name : intl.formatMessage({ id: 'devices.agent' }, { browser: name, os });
}

function lastActive(session: Session): number {
  return (session.last_used_at ?? session.created_at).getTime();
}

/** Revoking a session that is already gone has the outcome the reader wanted. */
async function revokeIfPresent(id: string): Promise<void> {
  try {
    await revokeSession(id);
  } catch (error) {
    if (!isProblem(error, 404)) throw error;
  }
}

type Revoked = { device: string } | { count: number };

export function Devices() {
  const intl = useIntl();
  const sessions = useSessions();
  const queryClient = useQueryClient();
  const reasonId = useId();

  // The focused button goes with the device it signed out, so the confirmation takes focus.
  // A new key per action re-mounts it, so a second revoke is announced too.
  const [revoked, setRevoked] = useState<{ key: number; what: Revoked }>();
  const announce = (what: Revoked) => setRevoked((previous) => ({ key: (previous?.key ?? 0) + 1, what }));

  const revokeOthers = useMutation({
    mutationFn: (ids: string[]) => Promise.all(ids.map(revokeIfPresent)),
    onSuccess: (_, ids) => announce({ count: ids.length }),
    onSettled: () => queryClient.invalidateQueries({ queryKey: queryKeys.sessions }),
  });

  let body;
  if (sessions.data === undefined) {
    body = sessions.isError ? (
      <ErrorState error={sessions.error} onRetry={() => void sessions.refetch()} />
    ) : (
      <Loading />
    );
  } else {
    // This device first, then the most recently used.
    const sorted = sessions.data.toSorted(
      (a, b) => Number(b.current) - Number(a.current) || lastActive(b) - lastActive(a),
    );
    const others = sorted.filter((session) => !session.current).map((session) => session.id);
    body = (
      <>
        {/* A failed background refresh keeps the list it had. */}
        {sessions.isError && <ErrorState error={sessions.error} onRetry={() => void sessions.refetch()} />}
        {revoked !== undefined && (
          <Notice key={revoked.key} tone="success" live="status" focusOnMount>
            <p>
              {'device' in revoked.what
                ? intl.formatMessage({ id: 'devices.revoked' }, { device: revoked.what.device })
                : intl.formatMessage({ id: 'devices.revokedOthers' }, { count: revoked.what.count })}
            </p>
          </Notice>
        )}
        <Section title={intl.formatMessage({ id: 'devices.listHeading' })}>
          {sorted.length === 0 ? (
            <p>{intl.formatMessage({ id: 'devices.empty' })}</p>
          ) : (
            <ul className={styles.list}>
              {sorted.map((session) => (
                <SessionItem key={session.id} session={session} onRevoked={(device) => announce({ device })} />
              ))}
            </ul>
          )}
        </Section>
        {revokeOthers.isError && (
          <ErrorState error={revokeOthers.error} onRetry={() => revokeOthers.mutate(others)} />
        )}
        <Actions>
          {others.length > 0 ? (
            <Button variant="primary" disabled={revokeOthers.isPending} onClick={() => revokeOthers.mutate(others)}>
              {revokeOthers.isPending
                ? intl.formatMessage({ id: 'devices.signingOut' })
                : intl.formatMessage({ id: 'devices.revokeOthers' }, { count: others.length })}
            </Button>
          ) : (
            <Button variant="primary" aria-disabled="true" aria-describedby={reasonId}>
              {intl.formatMessage({ id: 'devices.revokeOthersNone' })}
            </Button>
          )}
        </Actions>
        {others.length === 0 && (
          <p id={reasonId} className={styles.muted}>
            {intl.formatMessage({ id: 'devices.noOthers' })}
          </p>
        )}
      </>
    );
  }

  return (
    <Page title={intl.formatMessage({ id: 'devices.title' })} lead={intl.formatMessage({ id: 'devices.lead' })}>
      {body}
    </Page>
  );
}

function SessionItem({ session, onRevoked }: { session: Session; onRevoked: (device: string) => void }) {
  const intl = useIntl();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const name = deviceName(intl, session);
  const date = (value: Date) => intl.formatDate(value, { dateStyle: 'medium', timeStyle: 'short' });

  const revoke = useMutation({
    mutationFn: () => revokeIfPresent(session.id),
    onSuccess: () => {
      onRevoked(name);
      return queryClient.invalidateQueries({ queryKey: queryKeys.sessions });
    },
  });

  const signOut = useMutation({
    mutationFn: async () => {
      try {
        await deleteCurrentSession();
      } catch (error) {
        // 401: the session had already ended, which is where signing out leads anyway.
        if (!isProblem(error, 401)) throw error;
      }
    },
    onSuccess: () => {
      queryClient.clear();
      const state: SignInReason = { reason: 'signed-out' };
      void navigate('/sign-in', { replace: true, state });
    },
  });

  const action = session.current ? signOut : revoke;

  return (
    <li className={`${styles.item ?? ''} ${session.current ? (styles.itemCurrent ?? '') : ''}`}>
      <div className={styles.itemHeader}>
        <h3 className={styles.itemTitle}>{name}</h3>
        {session.current && <span className={styles.badge}>{intl.formatMessage({ id: 'devices.thisDevice' })}</span>}
      </div>
      <div className={styles.meta}>
        <p>{intl.formatMessage({ id: 'devices.signedIn' }, { date: date(session.created_at) })}</p>
        {session.last_used_at != null && (
          <p>{intl.formatMessage({ id: 'devices.lastActive' }, { date: date(session.last_used_at) })}</p>
        )}
      </div>
      {action.isError && <ErrorState error={action.error} onRetry={() => action.mutate()} />}
      <Actions>
        {session.current ? (
          <Button disabled={signOut.isPending} onClick={() => signOut.mutate()}>
            {intl.formatMessage({ id: signOut.isPending ? 'devices.signingOut' : 'devices.signOut' })}
          </Button>
        ) : (
          <Button
            variant="dangerQuiet"
            disabled={revoke.isPending}
            aria-label={revoke.isPending ? undefined : intl.formatMessage({ id: 'devices.revokeNamed' }, { device: name })}
            onClick={() => revoke.mutate()}
          >
            {intl.formatMessage({ id: revoke.isPending ? 'devices.revoking' : 'devices.revoke' })}
          </Button>
        )}
      </Actions>
    </li>
  );
}
