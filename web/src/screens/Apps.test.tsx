import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { empty, json, mockApi, problem } from '../test/api';
import { renderApp } from '../test/render';

const list = 'GET /api/auth/v1/clients';
const create = 'POST /api/auth/v1/clients';
const app = {
  id: 'client-1',
  name: 'Acme Notes',
  redirect_uris: ['https://notes.example/callback'],
  scopes: ['user:read'],
  created_at: '2026-10-20T10:00:00Z',
};

describe('OAuth apps', () => {
  it('creates an app and shows its secret once, with a warning and a copy button', async () => {
    const api = mockApi({
      [list]: [json(200, { clients: [], limit: 20, offset: 0 }), json(200, { clients: [app], limit: 20, offset: 0 })],
      [create]: json(201, { client: app, client_secret: 's3cr3t-value' }),
    });
    const { user } = renderApp('/apps');

    expect(await screen.findByText("You haven't registered any apps yet.")).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Register an app' }));
    await user.type(screen.getByLabelText('Name'), 'Acme Notes');
    await user.type(
      screen.getByLabelText('Redirect URIs'),
      'https://notes.example/callback{Enter}{Enter}https://notes.example/callback',
    );
    await user.click(screen.getByRole('checkbox', { name: 'user:read' }));
    await user.click(screen.getByRole('button', { name: 'Create app' }));

    expect(await screen.findByText("Copy the client secret now. You won't be able to see it again.")).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 2, name: 'Acme Notes is ready' })).toHaveFocus();
    expect(screen.getByText('s3cr3t-value')).toBeInTheDocument();
    expect(api.callsTo(create)[0]?.body).toEqual({
      name: 'Acme Notes',
      redirect_uris: ['https://notes.example/callback'],
      scopes: ['user:read'],
    });

    await user.click(screen.getByRole('button', { name: 'Copy client secret' }));
    expect(await navigator.clipboard.readText()).toBe('s3cr3t-value');

    await user.click(screen.getByRole('button', { name: "I've saved the secret" }));
    expect(screen.queryByText('s3cr3t-value')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Register an app' })).toHaveFocus();
    expect(await screen.findByRole('heading', { level: 3, name: 'Acme Notes' })).toBeInTheDocument();
  });

  it('checks the form before sending it', async () => {
    const api = mockApi({ [list]: json(200, { clients: [], limit: 20, offset: 0 }) });
    const { user } = renderApp('/apps');

    await user.click(await screen.findByRole('button', { name: 'Register an app' }));
    await user.type(screen.getByLabelText('Redirect URIs'), 'notes.example/callback');
    await user.click(screen.getByRole('button', { name: 'Create app' }));

    expect(screen.getByLabelText('Name')).toHaveAccessibleDescription(/Enter a name\./);
    expect(screen.getByLabelText('Name')).toHaveFocus();
    expect(screen.getByLabelText('Redirect URIs')).toHaveAccessibleDescription(
      /“notes.example\/callback” isn’t a full http or https URL\./,
    );
    expect(api.callsTo(create)).toHaveLength(0);
  });

  it("puts the server's field errors on their fields", async () => {
    mockApi({
      [list]: json(200, { clients: [], limit: 20, offset: 0 }),
      [create]: problem(400, { errors: { 'redirect_uris[0]': 'must use https outside localhost' } }),
    });
    const { user } = renderApp('/apps');

    await user.click(await screen.findByRole('button', { name: 'Register an app' }));
    await user.type(screen.getByLabelText('Name'), 'Acme');
    await user.type(screen.getByLabelText('Redirect URIs'), 'http://notes.example/callback');
    await user.click(screen.getByRole('button', { name: 'Create app' }));

    expect(await screen.findByText('must use https outside localhost')).toBeInTheDocument();
    expect(screen.getByLabelText('Redirect URIs')).toHaveAccessibleDescription(/must use https outside localhost/);
    expect(screen.getByLabelText('Redirect URIs')).toHaveFocus();
  });

  it('deletes an app only after confirming', async () => {
    const api = mockApi({
      [list]: [json(200, { clients: [app], limit: 20, offset: 0 }), json(200, { clients: [], limit: 20, offset: 0 })],
      'DELETE /api/auth/v1/clients/client-1': empty(204),
    });
    const { user } = renderApp('/apps');

    await user.click(await screen.findByRole('button', { name: 'Delete Acme Notes' }));
    const dialog = screen.getByRole('dialog', { name: 'Delete Acme Notes?' });
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }));
    expect(api.callsTo('DELETE /api/auth/v1/clients/client-1')).toHaveLength(0);
    expect(screen.getByRole('button', { name: 'Delete Acme Notes' })).toHaveFocus();

    await user.click(screen.getByRole('button', { name: 'Delete Acme Notes' }));
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete app' }));

    expect(await screen.findByText('Acme Notes was deleted.')).toBeInTheDocument();
    expect(await screen.findByText("You haven't registered any apps yet.")).toBeInTheDocument();
    expect(api.callsTo('DELETE /api/auth/v1/clients/client-1')).toHaveLength(1);
  });

  it('loads the next page when the first one is full', async () => {
    const second = { ...app, id: 'client-2', name: 'Acme Tasks' };
    const third = { ...app, id: 'client-3', name: 'Acme Mail' };
    const api = mockApi({
      [list]: json(200, { clients: [app, second], limit: 2, offset: 0 }),
      // A short page is the last one.
      'GET /api/auth/v1/clients?offset=2': json(200, { clients: [third], limit: 2, offset: 2 }),
    });
    const { user } = renderApp('/apps');

    await user.click(await screen.findByRole('button', { name: 'Show more apps' }));

    expect(await screen.findByRole('heading', { level: 3, name: 'Acme Mail' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 3, name: 'Acme Notes' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Show more apps' })).not.toBeInTheDocument();
    expect(api.callsTo('GET /api/auth/v1/clients?offset=2')).toHaveLength(1);
  });
});
