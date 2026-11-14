# Deploying Maestro

A Maestro is one server, orq, with the Maestro page served beside it, on
one host, and the machines it drives. The
[maestro](https://github.com/gustaleitcor/maestro) repo ties them together:
each part is a git submodule of it, and its compose files run them.

| Part | What it is | How it runs |
| --- | --- | --- |
| `maestro-orq` | the server: Go, SQLite, its own Podman for builds | a container that stays up, on port 3005 |
| `maestro-view` | the Maestro page: a Vue app | built once into `maestro-view/dist`, then served as files |
| `maestro-cli/book` | this book: mdBook, which `maestro` also carries | optionally built once into `maestro-cli/book/dist`, then served as files |
| Redis | sessions, and answers kept a few seconds | any Redis, Valkey or KeyDB |
| a reverse proxy | TLS, and one host for all of it | Caddy, nginx, … |

## Production

```sh
git clone --recurse-submodules https://github.com/gustaleitcor/maestro
cd maestro
cp .env.example .env   # the identity provider, MACHINE_SECRET, REDIS_URL
podman compose up --build -d
```

`orq` stays up as a server. `view` and `docs` build `maestro-view/dist` and
`maestro-cli/book/dist` and exit: the reverse proxy serves those directories
directly, not through a container.

The reverse proxy, on the page's host, must:

- send `/api/*` to orq (port 3005), so that `OIDC_REDIRECT_URL` reaches it
  and the session cookie is first-party;
- serve `maestro-cli/book/dist` at `/docs/`, which the page's Docs link and
  `maestro docs --web` open (`maestro docs` alone opens the copy `maestro`
  carries, and needs none of this);
- serve `maestro-view/dist` for everything else.

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

To update, pull, then build again; a container that already exists is only
replaced with `--force-recreate`:

```sh
git pull --recurse-submodules
podman compose up --build -d --force-recreate orq view docs
```

orq keeps everything about runs in its database, so it can be restarted in
the middle of one: see [What orq keeps](retention.md).

## What orq needs

- **Its Podman**, through `PODMAN_SOCKET`, to build images. The compose file
  mounts the host user's socket.
- **Two volumes:** `/data` for the SQLite database, and `/maestro/data` for
  what runs leave behind, which grows with every run.
- **Redis**, at `REDIS_URL`.
- **An identity provider**: see [The identity provider](identity.md).
- **`MACHINE_SECRET`**, at least 32 characters, which encrypts the machines'
  SSH keys: see [Adding machines](machines.md).

Every variable is in [Environment variables](../reference/env.md).

## The page and the book

The page is built with `VITE_ORQ_URL` (where orq is, the same host in
production) and optionally `VITE_IDP_NAME`, the sign-in button's label. It
has no server of its own, nor does the book.

## A development stack

```sh
git clone --recurse-submodules https://github.com/gustaleitcor/maestro
podman compose -f docker-compose-dev.yml up --build
```

The dev stack runs its own throwaway identity provider (Dex) and Valkey. Open
the page with `cd maestro-view && npm run dev` (with
`VIEW_URL=http://localhost:5173` exported before starting the stack), or
serve `maestro-view/dist` on port 8080, and sign in as `dev@example.com`
with the password `password`. `mdbook serve maestro-cli/book` serves this
book on port 3000 as you edit it; `MAESTRO_DOCS_URL=http://localhost:3000`
points `maestro docs` at it. See [Contributing](../contributing.md#this-book).
