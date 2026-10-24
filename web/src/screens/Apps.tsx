import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useEffect, useId, useRef, useState, type SubmitEvent } from 'react';
import { useIntl } from 'react-intl';
import { isProblem } from '../api/describe';
import { createClient, deleteClient, type NewClient } from '../api/endpoints';
import { queryKeys, useClients } from '../api/queries';
import type { Client, CreatedClient } from '../api/schemas';
import { grantableScopes, scopeDescriptions } from '../i18n/scopes';
import { Actions, Button } from '../ui/Button';
import { ConfirmDialog } from '../ui/ConfirmDialog';
import { ErrorState } from '../ui/ErrorState';
import { Checkbox, CheckboxGroup, TextAreaField, TextField } from '../ui/Field';
import { Loading } from '../ui/Loading';
import { Notice } from '../ui/Notice';
import { Page, Section } from '../ui/Page';
import { SecretValue } from '../ui/SecretValue';
import { useFocusFirstInvalid } from '../ui/useFocusFirstInvalid';
import styles from './screens.module.css';

type Mode = { kind: 'list' } | { kind: 'form' } | { kind: 'created'; created: CreatedClient };

export function Apps() {
  const intl = useIntl();
  const [mode, setMode] = useState<Mode>({ kind: 'list' });
  const [deleted, setDeleted] = useState<string>();

  // The form or the secret held focus. Leaving it lands on the button that opened it, not the top of the page.
  const newApp = useRef<HTMLButtonElement>(null);
  const focusNewApp = useRef(false);
  useEffect(() => {
    if (mode.kind !== 'list' || !focusNewApp.current) return;
    focusNewApp.current = false;
    newApp.current?.focus();
  }, [mode]);
  function backToList() {
    focusNewApp.current = true;
    setMode({ kind: 'list' });
  }

  return (
    <Page title={intl.formatMessage({ id: 'apps.title' })} lead={intl.formatMessage({ id: 'apps.lead' })}>
      {mode.kind === 'list' && (
        <Actions>
          <Button
            ref={newApp}
            variant="primary"
            onClick={() => {
              setDeleted(undefined);
              setMode({ kind: 'form' });
            }}
          >
            {intl.formatMessage({ id: 'apps.new' })}
          </Button>
        </Actions>
      )}
      {mode.kind === 'form' && (
        <CreateAppForm
          onCreated={(created) => setMode({ kind: 'created', created })}
          onCancel={backToList}
        />
      )}
      {mode.kind === 'created' && <CreatedApp created={mode.created} onDone={backToList} />}
      {deleted !== undefined && (
        <Notice tone="success" live="status">
          <p>{intl.formatMessage({ id: 'apps.deleted' }, { name: deleted })}</p>
        </Notice>
      )}
      <AppList onDeleted={setDeleted} />
    </Page>
  );
}

interface FormErrors {
  name?: string;
  redirects?: string;
  scopes?: string;
}

function parseRedirects(text: string): string[] {
  return [...new Set(text.split('\n').map((line) => line.trim()).filter((line) => line !== ''))];
}

function isHTTPURL(value: string): boolean {
  try {
    const url = new URL(value);
    return url.protocol === 'https:' || url.protocol === 'http:';
  } catch {
    return false;
  }
}

function CreateAppForm({ onCreated, onCancel }: { onCreated: (created: CreatedClient) => void; onCancel: () => void }) {
  const intl = useIntl();
  const queryClient = useQueryClient();
  const [name, setName] = useState('');
  const [redirects, setRedirects] = useState('');
  const [scopes, setScopes] = useState<string[]>([]);
  const [invalid, setInvalid] = useState<FormErrors>({});

  const create = useMutation({
    mutationFn: (client: NewClient) => createClient(client),
    onSuccess: (created) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.clients });
      onCreated(created);
    },
  });

  const { form, submitted } = useFocusFirstInvalid(create.error);

  function validate(client: NewClient): FormErrors {
    const errors: FormErrors = {};
    if (client.name === '') errors.name = intl.formatMessage({ id: 'apps.form.nameRequired' });
    const bad = client.redirect_uris.find((uri) => !isHTTPURL(uri));
    if (client.redirect_uris.length === 0) {
      errors.redirects = intl.formatMessage({ id: 'apps.form.redirectsRequired' });
    } else if (bad !== undefined) {
      errors.redirects = intl.formatMessage({ id: 'apps.form.redirectInvalid' }, { uri: bad });
    }
    return errors;
  }

  function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const client: NewClient = { name: name.trim(), redirect_uris: parseRedirects(redirects), scopes };
    const errors = validate(client);
    setInvalid(errors);
    submitted();
    if (Object.keys(errors).length === 0) create.mutate(client);
  }

  // The server names fields as the request body does; redirect URI errors may carry an index.
  const server = isProblem(create.error, 400) ? create.error.fieldErrors : {};
  const fieldFor = (key: string) =>
    key === 'name' ? 'name' : key.startsWith('redirect_uris') ? 'redirects' : key.startsWith('scopes') ? 'scopes' : undefined;
  const serverErrors: FormErrors = {};
  const otherErrors: string[] = [];
  for (const [key, problem] of Object.entries(server)) {
    const field = fieldFor(key);
    if (field === undefined) otherErrors.push(`${key}: ${problem}`);
    else serverErrors[field] ??= problem;
  }
  const errors = Object.keys(invalid).length > 0 ? invalid : serverErrors;
  const formError = create.isError && Object.keys(server).length === 0 ? create.error : undefined;

  return (
    <Section title={intl.formatMessage({ id: 'apps.form.heading' })}>
      <form ref={form} className={styles.form} noValidate onSubmit={submit}>
        <TextField
          label={intl.formatMessage({ id: 'apps.form.name' })}
          hint={intl.formatMessage({ id: 'apps.form.nameHint' })}
          name="name"
          autoComplete="off"
          // The form appears because the reader asked for it, so focus goes to its first field.
          autoFocus
          required
          value={name}
          error={errors.name}
          onChange={(event) => setName(event.target.value)}
        />
        <TextAreaField
          label={intl.formatMessage({ id: 'apps.form.redirects' })}
          hint={intl.formatMessage({ id: 'apps.form.redirectsHint' })}
          name="redirect_uris"
          autoComplete="off"
          spellCheck={false}
          autoCapitalize="none"
          placeholder="https://app.example.com/callback"
          required
          rows={3}
          value={redirects}
          error={errors.redirects}
          onChange={(event) => setRedirects(event.target.value)}
        />
        <CheckboxGroup
          legend={intl.formatMessage({ id: 'apps.form.scopes' })}
          hint={intl.formatMessage({ id: 'apps.form.scopesHint' })}
          error={errors.scopes}
        >
          {grantableScopes.map((scope) => (
            <Checkbox
              key={scope}
              name="scopes"
              value={scope}
              label={scope}
              description={intl.formatMessage({ id: scopeDescriptions[scope] })}
              checked={scopes.includes(scope)}
              onChange={(event) =>
                setScopes((current) =>
                  event.target.checked ? [...current, scope] : current.filter((value) => value !== scope),
                )
              }
            />
          ))}
        </CheckboxGroup>
        {otherErrors.length > 0 && (
          <Notice tone="danger" live="alert" title={intl.formatMessage({ id: 'apps.form.problems' })}>
            <ul className={styles.plainList}>
              {otherErrors.map((problem) => (
                <li key={problem}>{problem}</li>
              ))}
            </ul>
          </Notice>
        )}
        {formError !== undefined && <ErrorState error={formError} />}
        <Actions>
          <Button type="submit" variant="primary" disabled={create.isPending}>
            {intl.formatMessage({ id: create.isPending ? 'apps.form.creating' : 'apps.form.submit' })}
          </Button>
          <Button onClick={onCancel} disabled={create.isPending}>
            {intl.formatMessage({ id: 'common.cancel' })}
          </Button>
        </Actions>
      </form>
    </Section>
  );
}

function CreatedApp({ created, onDone }: { created: CreatedClient; onDone: () => void }) {
  const intl = useIntl();
  return (
    <Section title={intl.formatMessage({ id: 'apps.created.heading' }, { name: created.client.name })} focusOnMount>
      <Notice tone="warning" live="alert">
        <p>{intl.formatMessage({ id: 'apps.created.warning' })}</p>
      </Notice>
      <SecretValue
        label={intl.formatMessage({ id: 'apps.created.clientId' })}
        copyLabel={intl.formatMessage({ id: 'apps.created.copyClientId' })}
        value={created.client.id}
      />
      <SecretValue
        label={intl.formatMessage({ id: 'apps.created.secret' })}
        copyLabel={intl.formatMessage({ id: 'apps.created.copySecret' })}
        value={created.client_secret}
      />
      <Actions>
        <Button variant="primary" onClick={onDone}>
          {intl.formatMessage({ id: 'apps.created.done' })}
        </Button>
      </Actions>
    </Section>
  );
}

function AppList({ onDeleted }: { onDeleted: (name: string) => void }) {
  const intl = useIntl();
  const queryClient = useQueryClient();
  const clients = useClients();
  const headingId = useId();
  const heading = useRef<HTMLHeadingElement>(null);
  const [pending, setPending] = useState<Client>();

  // The deleted row held focus. Once the dialog has closed, land on the list rather than the top of the page.
  const focusListOnClose = useRef(false);
  useEffect(() => {
    if (pending !== undefined || !focusListOnClose.current) return;
    focusListOnClose.current = false;
    heading.current?.focus();
  }, [pending]);

  const remove = useMutation({
    mutationFn: async (client: Client) => {
      try {
        await deleteClient(client.id);
      } catch (error) {
        // Already gone is the outcome the reader asked for.
        if (!isProblem(error, 404)) throw error;
      }
    },
    onSuccess: (_, client) => {
      focusListOnClose.current = true;
      setPending(undefined);
      void queryClient.invalidateQueries({ queryKey: queryKeys.clients });
      onDeleted(client.name);
    },
  });

  let body;
  if (clients.data === undefined) {
    body = clients.isError ? (
      <ErrorState error={clients.error} onRetry={() => void clients.refetch()} />
    ) : (
      <Loading />
    );
  } else {
    // Data stays on screen when a later page or a background refresh fails.
    const all = clients.data.pages.flatMap((page) => page.clients);
    const retry = clients.isFetchNextPageError ? clients.fetchNextPage : clients.refetch;
    body = (
      <>
        {all.length === 0 ? (
          <p>{intl.formatMessage({ id: 'apps.list.empty' })}</p>
        ) : (
          <ul className={styles.list}>
            {all.map((client) => (
              <AppItem key={client.id} client={client} onDelete={() => setPending(client)} />
            ))}
          </ul>
        )}
        {clients.isError && <ErrorState error={clients.error} onRetry={() => void retry()} />}
        {clients.hasNextPage && (
          <Actions>
            <Button disabled={clients.isFetchingNextPage} onClick={() => void clients.fetchNextPage()}>
              {intl.formatMessage({ id: clients.isFetchingNextPage ? 'common.loading' : 'apps.list.more' })}
            </Button>
          </Actions>
        )}
      </>
    );
  }

  return (
    <section aria-labelledby={headingId} className={styles.form}>
      <h2 id={headingId} ref={heading} tabIndex={-1}>
        {intl.formatMessage({ id: 'apps.list.heading' })}
      </h2>
      {body}
      <ConfirmDialog
        open={pending !== undefined}
        title={intl.formatMessage({ id: 'apps.delete.title' }, { name: pending?.name ?? '' })}
        confirmLabel={intl.formatMessage({ id: 'apps.delete.confirm' })}
        pendingLabel={intl.formatMessage({ id: 'apps.delete.deleting' })}
        pending={remove.isPending}
        onConfirm={() => {
          if (pending !== undefined) remove.mutate(pending);
        }}
        onCancel={() => {
          setPending(undefined);
          remove.reset();
        }}
      >
        <p>{intl.formatMessage({ id: 'apps.delete.body' })}</p>
        {remove.isError && <ErrorState error={remove.error} />}
      </ConfirmDialog>
    </section>
  );
}

function AppItem({ client, onDelete }: { client: Client; onDelete: () => void }) {
  const intl = useIntl();
  return (
    <li className={styles.item}>
      <h3 className={styles.itemTitle}>{client.name}</h3>
      <dl className={styles.details}>
        <dt>{intl.formatMessage({ id: 'apps.list.clientId' })}</dt>
        <dd>
          <code className={styles.code}>{client.id}</code>
        </dd>
        <dt>{intl.formatMessage({ id: 'apps.list.redirects' })}</dt>
        <dd>
          <ul className={styles.plainList}>
            {client.redirect_uris.map((uri) => (
              <li key={uri}>
                <code className={styles.code}>{uri}</code>
              </li>
            ))}
          </ul>
        </dd>
        <dt>{intl.formatMessage({ id: 'apps.list.scopes' })}</dt>
        <dd>
          {client.scopes.length === 0 ? (
            intl.formatMessage({ id: 'apps.list.noScopes' })
          ) : (
            <ul className={styles.tags}>
              {client.scopes.map((scope) => (
                <li key={scope} className={styles.tag}>
                  {scope}
                </li>
              ))}
            </ul>
          )}
        </dd>
      </dl>
      {client.created_at !== undefined && (
        <p className={styles.muted}>
          {intl.formatMessage(
            { id: 'apps.list.created' },
            { date: intl.formatDate(client.created_at, { dateStyle: 'medium' }) },
          )}
        </p>
      )}
      <Actions>
        <Button
          variant="dangerQuiet"
          aria-label={intl.formatMessage({ id: 'apps.list.deleteNamed' }, { name: client.name })}
          onClick={onDelete}
        >
          {intl.formatMessage({ id: 'apps.list.delete' })}
        </Button>
      </Actions>
    </li>
  );
}
