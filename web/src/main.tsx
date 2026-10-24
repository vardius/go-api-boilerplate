import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { createBrowserRouter } from 'react-router';
import { App } from './app/App';
import { createQueryClient, sessionExpiredHandler } from './app/queryClient';
import { routes } from './app/routes';
import './styles/tokens.css';
import './styles/base.css';

const root = document.getElementById('root');
if (root === null) throw new Error('index.html has no #root element.');

const router = createBrowserRouter(routes);
const queryClient = createQueryClient({ onSessionExpired: sessionExpiredHandler(router) });

createRoot(root).render(
  <StrictMode>
    <App router={router} queryClient={queryClient} />
  </StrictMode>,
);
