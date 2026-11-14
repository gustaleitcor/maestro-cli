# The identity provider

Maestro trusts exactly one identity provider per deployment, over standard
[OpenID Connect](https://openid.net/developers/how-connect-works/). orq sends
the browser to the provider, checks the ID token it gets back, and then keeps
its own session (the `maestro_session` cookie, kept in Redis). Nothing in
Maestro is specific to any one provider.

## Registering Maestro

Register Maestro with your provider as a confidential client using the
authorization code flow, with the redirect URI `<orq>/api/auth/callback`
(the page's host, in production: `https://maestro.example.org/api/auth/callback`),
then configure orq:

| Variable | Default | |
| --- | --- | --- |
| `OIDC_ISSUER` | required | Issuer URL; must serve `/.well-known/openid-configuration` |
| `OIDC_CLIENT_ID` | required | |
| `OIDC_CLIENT_SECRET` | required | |
| `OIDC_REDIRECT_URL` | required | `<orq>/api/auth/callback`, as registered with the provider |
| `OIDC_SCOPES` | `openid email profile` | Space-separated |
| `OIDC_ROLES_CLAIM` | `roles` | ID token claim holding the user's roles (a string or a list); e.g. `groups` |
| `OIDC_ALLOWED_ROLES` | empty | Comma-separated. Empty lets in anyone the provider authenticates |
| `VIEW_URL` | first of `ALLOWED_ORIGINS` | Where the Maestro page is served; logins return there |
| `REDIS_URL` | required | `redis://[:password@]host:port/db` of a Redis, Valkey or KeyDB, where sessions are kept |
| `SESSION_TTL` | `12h` | How long a Maestro session lasts |
| `COOKIE_SECURE` | `true` (`false` with `MODE=dev`) | Set `false` only when serving over plain http |

The page only needs `VITE_ORQ_URL`, and optionally `VITE_IDP_NAME` for the
sign-in button's label.

## Administrators

A user is an administrator when the provider gives them one of
`OIDC_ADMIN_ROLES` (by default `ADMIN`) in the roles claim, or their email is
in `ADMIN_EMAILS`. Administrators add machines, manage groups and the
scheduler's settings, and see everyone's runs and containers.

## Providers

- **SaciAuthService** (auth.logsad.com): what logsad.com runs; tested end to
  end.
- **Dex**: what the dev stack runs.
- **Keycloak, Authentik, Zitadel, …**: any provider with OIDC discovery, the
  authorization code flow, PKCE (S256) and RS256/ES256-signed ID tokens
  should work, but these have not been tried yet. Point `OIDC_ROLES_CLAIM`
  at whichever claim your provider puts roles or groups in.

## Sessions and keys

Signing out of Maestro ends the Maestro session only; it does not sign the
user out of the provider, and signing out of the provider does not end a
Maestro session that is already open.

The CLI doesn't use sessions. `maestro login` has the user approve a code on
the page, which hands the CLI a Maestro key of its own, kept until revoked.
