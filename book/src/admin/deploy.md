# Deploying Maestro

A Maestro deployment is one host that runs the server, orq, and serves the
Maestro page beside it, plus the machines orq drives. The
[maestro](https://github.com/gustaleitcor/maestro) repository puts the parts
together. Each part is a git submodule of it, and its compose files run
them.

| Part | What it is | How it runs |
| --- | --- | --- |
| `maestro-orq` | The server, written in Go, with SQLite and its own Podman for builds | A container that stays up, on port 3005 |
| `maestro-view` | The Maestro page, a Vue app | Built once into `maestro-view/dist`, then served as files |
| `maestro-cli/book` | This book, which `maestro` also carries | Optionally built once into `maestro-cli/book/dist`, then served as files |
| Redis | Sessions, and answers cached for a few seconds | Any Redis, Valkey or KeyDB |
| A reverse proxy | TLS, and one host name for everything | Caddy, nginx or another |

## Production

```sh
git clone --recurse-submodules https://github.com/gustaleitcor/maestro
cd maestro
cp .env.example .env   # the identity provider, MACHINE_SECRET, REDIS_URL
podman compose up --build -d
```

The `orq` service stays up as the server. The `view` and `docs` services
build `maestro-view/dist` and `maestro-cli/book/dist` and then exit. The
reverse proxy serves those two directories itself, with no container in
between.

On the page's host, the reverse proxy has three jobs.

1. Send `/api/*` to orq on port 3005. This lets `OIDC_REDIRECT_URL` reach
   orq and makes the session cookie first-party.
2. Serve `maestro-cli/book/dist` at `/docs/`. The page's Docs link and
   `maestro docs --web` open that copy. `maestro docs` without `--web` opens
   the copy inside the CLI and needs none of this.
3. Serve `maestro-view/dist` for every other path.

With Caddy:

```caddyfile
maestro.example.org {
	handle /api/* {
		reverse_proxy localhost:3005
	}
	handle_path /docs/* {
		root * /srv/maestro/maestro-cli/book/dist
		file_server
	}
	handle {
		root * /srv/maestro/maestro-view/dist
		try_files {path} /index.html
		file_server
	}
}
```

## Updating

Pull, then build again. Compose replaces a container that already exists
only when you pass `--force-recreate`:

```sh
git pull --recurse-submodules
podman compose up --build -d --force-recreate orq view docs
```

orq keeps everything about runs in its database, so you can restart it in
the middle of a run. [What orq keeps](retention.md) describes what happens
to the lines that were running.

## What orq needs

| Need | Detail |
| --- | --- |
| Its own Podman | Reached through `PODMAN_SOCKET`, to build images. The compose file mounts the host user's socket. |
| Two volumes | `/data` for the SQLite database, and `/maestro/data` for the logs and outputs of runs. The second grows with every run. |
| Redis | At `REDIS_URL` |
| An identity provider | See [The identity provider](identity.md). |
| `MACHINE_SECRET` | At least 32 characters. It encrypts the machines' SSH keys. See [Adding machines](machines.md). |

[Environment variables](../reference/env.md) lists every variable.

## The page and the book

The page is built with `VITE_ORQ_URL`, the address of orq, which in
production is the page's own host. `VITE_IDP_NAME` optionally sets the label
of the sign-in button. Neither the page nor the book has a server of its
own.

## A development stack

```sh
git clone --recurse-submodules https://github.com/gustaleitcor/maestro
podman compose -f docker-compose-dev.yml up --build
```

The development stack runs its own throwaway identity provider (Dex) and
Valkey. Sign in as `dev@example.com` with the password `password`.

There are two ways to open the page:

- Run `cd maestro-view && npm run dev`, with
  `VIEW_URL=http://localhost:5173` exported before you start the stack.
- Serve `maestro-view/dist` on port 8080.

`mdbook serve maestro-cli/book` serves this book on port 3000 and rebuilds
it as you edit. Set `MAESTRO_DOCS_URL=http://localhost:3000` to point
`maestro docs` at it. [Contributing](../contributing.md#this-book) has more
on working on the book.
