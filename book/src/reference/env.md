# Environment variables

## orq

| Variable | Default | |
| --- | --- | --- |
| `MODE` | empty | `dev` serves swagger and allows session cookies over plain http |
| `MAESTRO_ORQ_ADDR` | `:8081` | Where orq listens (the compose file uses `:3005`) |
| `MAESTRO_ORQ_DB` | `maestro-orq.db` | The SQLite database |
| `ALLOWED_ORIGINS` | required | Comma-separated origins of the Maestro page, for CORS |
| `VIEW_URL` | first of `ALLOWED_ORIGINS` | Where the Maestro page is served; logins return there |
| `PODMAN_SOCKET` | `unix:///run/podman/podman.sock` | orq's own Podman, which builds images |
| `BUILD_TIMEOUT` | `30m` | How long a build may take |
| `ALLOWED_FORGE_HOSTS` | `github.com,codeberg.org,gitlab.com` | The only hosts orq clones from, so it can't be pointed at hosts on its own network |
| `DATA_DIR` | `/maestro/data` | Where runs' logs and outputs are kept |
| `RUN_OUTPUT_MAX_MB` | `1024` | Logs and outputs kept per line; the rest is dropped, with a note |
| `RUN_PIDS` | `512` | Process limit per container; `0` for none |
| `REDIS_URL` | required | `redis://[:password@]host:port/db` of a Redis, Valkey or KeyDB |
| `SESSION_TTL` | `12h` | How long a session of the Maestro page lasts |
| `COOKIE_SECURE` | `true` (`false` with `MODE=dev`) | Set `false` only when serving over plain http |
| `OIDC_ISSUER` | required | Issuer URL; must serve `/.well-known/openid-configuration` |
| `OIDC_CLIENT_ID` | required | |
| `OIDC_CLIENT_SECRET` | required | |
| `OIDC_REDIRECT_URL` | required | `<orq>/api/auth/callback`, as registered with the provider |
| `OIDC_SCOPES` | `openid email profile` | Space-separated |
| `OIDC_ROLES_CLAIM` | `roles` | ID token claim holding the user's roles (a string or a list) |
| `OIDC_ALLOWED_ROLES` | empty | Comma-separated. Empty lets in anyone the provider authenticates |
| `OIDC_ADMIN_ROLES` | `ADMIN` | Comma-separated roles that make a user an administrator |
| `ADMIN_EMAILS` | empty | Comma-separated emails that are administrators regardless of roles |
| `MACHINE_SECRET` | empty | Encrypts the stored SSH keys; at least 32 characters. Empty disables machines |
| `MACHINE_PROXY_COMMANDS` | `false` | Allows machines reached through a proxy command, which runs on the orq server |

## The Maestro page (at build time)

| Variable | Default | |
| --- | --- | --- |
| `VITE_ORQ_URL` | `http://localhost:8081` | Where orq is |
| `VITE_IDP_NAME` | empty | The sign-in button's label |
| `VITE_DOCS_URL` | `/docs/` | Where the server's copy of this book is, for the sidebar's Docs |

## The CLI

| Variable | |
| --- | --- |
| `MAESTRO_ORQ_URL` | The Maestro server, instead of the one kept (`--server` comes first) |
| `MAESTRO_DOCS_URL` | Where `maestro docs` opens this book, instead of the copy it carries (or, with `--web`, the server's `/docs`) |
| `NO_COLOR` | Print without colours, as `--no-color` does |

## Inside each container

Set by orq, after the run's own `env`; see [Inside the
container](../run/container.md).

| Variable | |
| --- | --- |
| `MAESTRO_RUN`, `MAESTRO_LINE` | The run's number, and the line's in it, from 1 |
| `MAESTRO_CPUS` | Its `cpus` |
| `MAESTRO_MEMORY` | Its `memory`, in MB |
| `MAESTRO_OUTPUTS` | Its `outputs`, one per line |
