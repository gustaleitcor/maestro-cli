# Environment variables

## orq

| Variable | Default | Meaning |
| --- | --- | --- |
| `MODE` | empty | `dev` serves swagger to anyone and allows session cookies over plain http. Otherwise only someone signed in on the Maestro page sees swagger. |
| `MAESTRO_ORQ_ADDR` | `:8081` | Where orq listens. The compose file uses `:3005`. |
| `MAESTRO_ORQ_DB` | `maestro-orq.db` | The SQLite database |
| `DB_BACKUP_DIR` | empty | Where orq writes a copy of the database every day, as `maestro-orq-<date>.db`. When empty, orq makes no copies. See [Backing up the database](../admin/retention.md#backing-up-the-database). |
| `DB_BACKUPS_KEPT` | `7` | How many copies orq keeps in `DB_BACKUP_DIR`. It removes the oldest beyond that. `0` keeps every copy. |
| `ALLOWED_ORIGINS` | required | The origins of the Maestro page, separated by commas, for CORS |
| `VIEW_URL` | The first of `ALLOWED_ORIGINS` | Where the Maestro page is served. Sign-ins return there. |
| `PODMAN_SOCKET` | `unix:///run/podman/podman.sock` | orq's own Podman, which builds images |
| `BUILD_TIMEOUT` | `30m` | How long a build may take |
| `BUILDS_AT_ONCE` | `2` | How many builds each user may have running at once. `0` means no limit. |
| `ALLOWED_FORGE_HOSTS` | `github.com,codeberg.org,gitlab.com` | The only hosts orq clones from. The list keeps anyone from pointing orq at a host on its own network. |
| `DATA_DIR` | `/maestro/data` | Where the logs and outputs of runs are kept |
| `RUN_OUTPUT_MAX_MB` | `1024` | The logs and outputs kept for each line. orq drops the rest and leaves a note on the line. |
| `RUN_MAX_HOURS` | `168` (a week) | How long a line may run. orq stops a line that runs longer, keeps its logs and outputs, and ends it as `error`. `0` means no limit. |
| `USER_DATA_MAX_GB` | `0` | What all of a user's runs may keep, in GB. A user at the limit can't start a run until they delete one. `0` means no limit. |
| `RUN_PIDS` | `512` | The limit on processes in each container. `0` means no limit. |
| `REDIS_URL` | required | `redis://[:password@]host:port/db` of a Redis, Valkey or KeyDB |
| `SESSION_TTL` | `12h` | How long a session of the Maestro page lasts |
| `KEY_TTL` | `2160h` (90 days) | How long a Maestro key works after it was issued. `0` means keys never expire. |
| `COOKIE_SECURE` | `true`, or `false` with `MODE=dev` | Set it to `false` only when you serve over plain http. |
| `OIDC_ISSUER` | required | The issuer URL. It must serve `/.well-known/openid-configuration`. |
| `OIDC_CLIENT_ID` | required | |
| `OIDC_CLIENT_SECRET` | required | |
| `OIDC_REDIRECT_URL` | required | `<orq>/api/auth/callback`, as registered with the provider |
| `OIDC_SCOPES` | `openid email profile` | Separated by spaces |
| `OIDC_ROLES_CLAIM` | `roles` | The ID token claim that holds the user's roles, as a string or a list |
| `OIDC_ALLOWED_ROLES` | empty | Separated by commas. When empty, anyone the provider authenticates can sign in. |
| `OIDC_ADMIN_ROLES` | `ADMIN` | The roles that make a user an administrator, separated by commas |
| `ADMIN_EMAILS` | empty | Emails of users who are administrators whatever their roles, separated by commas |
| `MACHINE_SECRET` | empty | Encrypts the stored SSH keys. At least 32 characters. While it is empty, machines are turned off. |
| `MACHINE_PROXY_COMMANDS` | `false` | Allows machines reached through a proxy command, which runs on the orq server |

[Deploying Maestro](../admin/deploy.md) and
[The identity provider](../admin/identity.md) say which of these a new
deployment has to set.

## The Maestro page, at build time

| Variable | Default | Meaning |
| --- | --- | --- |
| `VITE_ORQ_URL` | `http://localhost:8081` | Where orq is |
| `VITE_IDP_NAME` | empty | The label of the sign-in button |

## The CLI

| Variable | Meaning |
| --- | --- |
| `MAESTRO_ORQ_URL` | The Maestro server to use in place of the one the CLI kept. `--server` wins over it. |
| `MAESTRO_DOCS_URL` | Where `maestro docs` opens this book, in place of the copy inside the CLI |
| `NO_COLOR` | Prints without colours, as `--no-color` does |

## Inside each container

orq sets these after the run's own `env`.
[Inside the container](../repo/container.md#what-maestro-tells-the-program)
shows how a program uses them.

| Variable | Value |
| --- | --- |
| `MAESTRO_RUN` | The run's number |
| `MAESTRO_LINE` | The line's number in the run, starting at 1 |
| `MAESTRO_CPUS` | The line's `cpus` |
| `MAESTRO_MEMORY` | The line's `memory`, in MB |
| `MAESTRO_OUTPUTS` | The line's `outputs`, one path on each text line |
