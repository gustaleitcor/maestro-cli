# Deploying Maestro

A Maestro deployment is one host that runs the server, orq, and serves the
Maestro page beside it, plus the machines orq drives. The
[maestro](https://github.com/gustaleitcor/maestro) repository puts the parts
together. Each part is a git submodule of it, and its compose files run
them.

| Part | What it is | How it runs |
| --- | --- | --- |
| `maestro-orq` | The server, written in Go, with SQLite and its own Podman for builds | A container that stays up, on port 3005 of 127.0.0.1 |
| `maestro-view` | The Maestro page, a Vue app | Built once into `maestro-view/dist`, then served as files |
| Redis | Sessions, and answers cached for a few seconds | Any Redis, Valkey or KeyDB |
| A reverse proxy | TLS, and one host name for everything | Caddy, nginx or another |

## Production

```sh
git clone --recurse-submodules https://github.com/gustaleitcor/maestro
cd maestro
cp .env.example .env   # the identity provider, MACHINE_SECRET, REDIS_URL
podman compose up --build -d
```

The `orq` service stays up as the server. The `view` service builds
`maestro-view/dist` and then exits. The reverse proxy serves that directory
itself, with no container in between.

On the page's host, the reverse proxy has two jobs.

1. Send `/api/*` to orq on port 3005. This lets `OIDC_REDIRECT_URL` reach
   orq and makes the session cookie first-party.
2. Serve `maestro-view/dist` for every other path.

With Caddy:

```caddyfile
maestro.example.org {
	handle /api/* {
		reverse_proxy localhost:3005
	}
	handle {
		root * /srv/maestro/maestro-view/dist
		try_files {path} /index.html
		file_server
	}
}
```

## Keeping rootless Podman running

The compose file runs Maestro with the Podman of the user who starts it,
without root. systemd stops that user's services, Podman's among them, when
their last session ends, and orq's container stops with them. Turn on
lingering for that user, so that their services keep running after they
sign out:

```sh
sudo loginctl enable-linger <user>
loginctl show-user <user> --property=Linger   # Linger=yes
```

## Updating

Pull, then build again. Compose replaces a container that already exists
only when you pass `--force-recreate`:

```sh
git pull --recurse-submodules
podman compose up --build -d --force-recreate orq view
```

orq keeps everything about runs in its database, so you can restart it in
the middle of a run. [What orq keeps](retention.md) describes what happens
to the lines that were running.

## What orq needs

| Need | Detail |
| --- | --- |
| Its own Podman | Reached through `PODMAN_SOCKET`, to build images. The compose file mounts the host user's socket. |
| Two volumes | `/data` for the SQLite database, and `/maestro/data` for the logs and outputs of runs. The second grows with every run. |
| Redis | At `REDIS_URL`, with a password. See [Redis](#redis). |
| An identity provider | See [The identity provider](identity.md). |
| `MACHINE_SECRET` | At least 32 characters. It encrypts the machines' SSH keys. See [Adding machines](machines.md). |

[Environment variables](../reference/env.md) lists every variable.

## The network

orq is reached through the reverse proxy and nothing else. The compose file
publishes its port 3005 on 127.0.0.1, so only the host itself reaches it.

A reverse proxy in a container of its own reaches the host through Podman's
bridge, at `host.containers.internal`, and a port published on 127.0.0.1
may not answer there. Try it on your host. If the proxy can't reach orq,
do one of two things:

- Set `ORQ_BIND` in `.env` to an address of the host that the proxy reaches
  and other hosts don't.
- Put the proxy and orq on one Podman network, have the proxy reach orq by
  its name, and remove the `ports` entry from the compose file.

The host also needs a firewall that lets in SSH and the ports the reverse
proxy listens on, usually 80 and 443, and nothing else. A port that rootful
Docker or Podman publishes can get past some firewalls' rules, so publish
orq on 127.0.0.1 even behind one. With firewalld:

```sh
firewall-cmd --permanent --add-service=http --add-service=https
firewall-cmd --permanent --list-services   # remove any other with --remove-service
firewall-cmd --reload
```

## Redis

Redis holds the sessions of the Maestro page, so whoever can write to it can
make a session and sign in as anyone, administrators included. Builds run on
the same host and may reach it. Hold it to two rules:

- Give it a password. Make one with `openssl rand -hex 32`, which needs no
  escaping in a URL. Start Redis with `--requirepass <password>`, or put
  `requirepass <password>` in its `redis.conf`. Then set
  `REDIS_URL=redis://:<password>@host:port/db` in `.env`. Unless
  `MODE=dev`, orq logs a warning at start while `REDIS_URL` has no password.
- Don't publish its port beyond the host. Other hosts must not reach it.

A Redis that other services share gets the same care from each of them. Any
one of them that publishes it, or lets its own users reach it, opens
Maestro's sessions too. When the Redis belongs to another compose project,
its owner sets the password there, and you then set it in `REDIS_URL`.

## The page and the book

The page is built with `VITE_ORQ_URL`, the address of orq, which in
production is the page's own host. `VITE_IDP_NAME` optionally sets the label
of the sign-in button. The page has no server of its own.

The deployment doesn't serve this book. `maestro` carries it, and
`maestro docs` opens it from your computer.

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
