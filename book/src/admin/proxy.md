# Machines behind a proxy

Some machines can't be reached over plain TCP, such as one behind
Cloudflare Access. orq reaches those through a proxy command, the way ssh
does with `ProxyCommand`. orq runs the command and speaks SSH over the
command's standard input and output. The host key is still pinned, and
everything else works as it does for any machine.

## Turning proxy commands on

A proxy command runs a program inside orq's container, so allowing them
lets any administrator run commands on the server. orq refuses proxy
commands unless `MACHINE_PROXY_COMMANDS=true`. While the setting is off, a
machine saved with a proxy command shows as unreachable and says why.

## Cloudflare Access

In the machine's form, set the host to the Access hostname, such as
`q1-ssh.example.com`, and set the proxy command to:

```text
cloudflared access ssh --hostname %h
```

orq replaces `%h`, `%p` and `%r` with the host, the port and the user, and
`%%` with a `%`. `cloudflared` is included in orq's image.

Under "Proxy environment", paste the Access service token:

```text
TUNNEL_SERVICE_TOKEN_ID=…
TUNNEL_SERVICE_TOKEN_SECRET=…
```

You can also paste a file of `export NAME=value` lines as it is, such as
one you source before running ssh.

orq stores the proxy environment encrypted with `MACHINE_SECRET`, as it
does the private key, and never sends it back. The page shows only the
names of the variables. Saving an edit with the field left empty keeps what
is stored.

The command runs with `PATH`, `HOME` and the variables you pasted. It gets
none of orq's own environment.

## What a proxy costs

orq doesn't keep connections open. Each connection from the runner, from a
check of the machine, or from `maestro top` starts its own `cloudflared`,
which adds between half a second and a second. `maestro top` opens one
every couple of seconds while someone is watching.

If `cloudflared` fails, the machine's error shows what it printed.
