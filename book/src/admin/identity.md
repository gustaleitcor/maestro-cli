# The identity provider

A Maestro deployment trusts one identity provider, which it talks to over
standard [OpenID Connect](https://openid.net/developers/how-connect-works/).
orq sends the browser to the provider, checks the ID token that comes back,
and then keeps a session of its own. The session is the `maestro_session`
cookie, stored in Redis. No part of Maestro depends on a particular
provider.

## Registering Maestro

Register Maestro with your provider as a confidential client that uses the
authorization code flow. The redirect URI is `<orq>/api/auth/callback`. In
production orq is on the page's host, so the URI looks like
`https://maestro.example.org/api/auth/callback`.

Then configure orq with these variables:

| Variable | Default | Meaning |
| --- | --- | --- |
| `OIDC_ISSUER` | required | The issuer URL. It must serve `/.well-known/openid-configuration`. |
| `OIDC_CLIENT_ID` | required | |
| `OIDC_CLIENT_SECRET` | required | |
| `OIDC_REDIRECT_URL` | required | `<orq>/api/auth/callback`, as registered with the provider |
| `OIDC_SCOPES` | `openid email profile` | Separated by spaces |
| `OIDC_ROLES_CLAIM` | `roles` | The ID token claim that holds the user's roles, as a string or a list. Some providers use `groups`. |
| `OIDC_ALLOWED_ROLES` | empty | Separated by commas. When empty, anyone the provider authenticates can sign in. |
| `VIEW_URL` | The first of `ALLOWED_ORIGINS` | Where the Maestro page is served. Sign-ins return there. |
| `REDIS_URL` | required | `redis://[:password@]host:port/db` of the Redis, Valkey or KeyDB that keeps sessions |
| `SESSION_TTL` | `12h` | How long a Maestro session lasts |
| `COOKIE_SECURE` | `true`, or `false` with `MODE=dev` | Set it to `false` only when you serve over plain http. |

The page needs `VITE_ORQ_URL`, and optionally `VITE_IDP_NAME` for the label
of the sign-in button.

## Administrators

A user is an administrator in either of two cases: the provider puts one of
`OIDC_ADMIN_ROLES` (by default `ADMIN`) in the user's roles claim, or the
user's email is in `ADMIN_EMAILS`.

Administrators add machines, manage groups and the scheduler's settings,
and see everyone's runs and containers.

## Providers that have been tried

| Provider | Status |
| --- | --- |
| SaciAuthService (auth.logsad.com) | Runs logsad.com's Maestro. Tested end to end. |
| Dex | Runs in the development stack. |
| Keycloak, Authentik, Zitadel and others | Untried. |

An untried provider should work if it has OIDC discovery, the authorization
code flow, PKCE with S256, and ID tokens signed with RS256 or ES256. Point
`OIDC_ROLES_CLAIM` at the claim your provider puts roles or groups in.

## Sessions and keys

Signing out of Maestro ends the Maestro session and leaves the user signed
in at the provider. Signing out at the provider leaves an open Maestro
session running until it expires.

The CLI doesn't use sessions. `maestro login` has the user approve a code on
the page, and orq then gives the CLI a Maestro key of its own. A key doesn't
expire, and Maestro has no screen for revoking one. `maestro logout` removes
the key from the user's computer.
