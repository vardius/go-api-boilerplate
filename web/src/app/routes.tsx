import { Navigate, type RouteObject } from 'react-router';
import { Apps } from '../screens/Apps';
import { CompleteSignIn } from '../screens/CompleteSignIn';
import { ConfirmEmail } from '../screens/ConfirmEmail';
import { Consent } from '../screens/Consent';
import { Devices } from '../screens/Devices';
import { NotFound } from '../screens/NotFound';
import { OAuthError } from '../screens/OAuthError';
import { Profile } from '../screens/Profile';
import { RouteError } from '../screens/RouteError';
import { SignIn } from '../screens/SignIn';
import { AccountLayout, PublicLayout, Root } from './Shell';

export const routes: RouteObject[] = [
  {
    element: <Root />,
    errorElement: <RouteError />,
    children: [
      {
        element: <AccountLayout />,
        children: [
          { path: '/', element: <Navigate to="/profile" replace /> },
          { path: '/profile', element: <Profile /> },
          { path: '/email/confirm', element: <ConfirmEmail /> },
          { path: '/devices', element: <Devices /> },
          { path: '/apps', element: <Apps /> },
        ],
      },
      {
        element: <PublicLayout />,
        children: [
          { path: '/sign-in', element: <SignIn /> },
          { path: '/sign-in/complete', element: <CompleteSignIn /> },
          { path: '/consent', element: <Consent /> },
          { path: '/oauth/error', element: <OAuthError /> },
          { path: '*', element: <NotFound /> },
        ],
      },
    ],
  },
];
