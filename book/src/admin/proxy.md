# Machines behind a proxy

A machine that can't be reached over plain TCP, such as one behind
Cloudflare Access, is reached through a proxy command, as with ssh's
`ProxyCommand`. orq runs the command and speaks SSH over its standard input
and output itself, so the host key is still pinned and everything else works
as for any machine.

## Cloudflare Access

In the machine's form, set the proxy command to

```text
cloudflared access ssh --hostname %h
```

with the host set to the Access hostname (e.g. `q1-ssh.example.com`). `%h`,
`%p` and `%r` become the host, port and user, and `%%` a `%`. cloudflared is
in orq's image.

Under *Proxy environment*, paste the Access service token as

```text
TUNNEL_SERVICE_TOKEN_ID=…
TUNNEL_SERVICE_TOKEN_SECRET=…
```

(a file of `export NAME=value` lines, such as one you source for ssh, can be
pasted as it is). Like the private key, it is stored encrypted with
`MACHINE_SECRET` and never sent back: the page shows only the names, and an
edit that leaves it empty keeps what is stored. The command gets only
`PATH`, `HOME` and these variables, none of orq's own environment.

## Turning them on

A proxy command runs a program inside orq's container, so it lets any
administrator run commands on the server. orq refuses them unless
`MACHINE_PROXY_COMMANDS=true`, and a machine saved with one shows as
unreachable, saying why, while it is off.

## Cost

orq doesn't keep connections open: each one, from the runner, a check or
`maestro top` (one every couple of seconds while someone watches it), starts
its own cloudflared, which adds half a second to a second to it. If
cloudflared fails, what it printed is in the machine's error.
