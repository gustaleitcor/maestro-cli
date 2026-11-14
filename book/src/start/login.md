# Signing in

Maestro signs you in through your lab's identity provider, the same account
you use for its other services. The Maestro page asks it directly; the CLI
asks the page.

## The Maestro page

Open the Maestro page and press the sign-in button: you go to the identity
provider, and come back signed in. The session lasts 12 hours unless your
administrator set otherwise.

Signing out of Maestro ends the Maestro session only. It doesn't sign you
out of the identity provider, and signing out of the provider doesn't end a
Maestro session already open.

## The CLI

```sh
maestro login
```

The CLI has no server of its own. The first time a command needs one and
nothing names it, `maestro` asks for the server's URL (`https://` is assumed
when it says no scheme) and keeps it. Then `maestro login` prints a short
code and a link to the Maestro page; open the link, sign in there if you
aren't, and approve the code. The CLI gets a Maestro key of its own.

The link is not opened for you, so it can be followed in any browser, on
that machine or another: signing in on a server you reach over SSH works the
same way.

The key is kept in the system keyring. If yours asks for a password, or there
is none (as on a server), `maestro login --store-in-file` keeps secrets in
the CLI's config file instead, readable only by you.

`maestro` alone says who is signed in, where, and to which forges.

## More than one Maestro

```sh
maestro login --server https://maestro.example.org
```

signs in to another Maestro, which every command then talks to. `--server`
on any command talks to a server just that once, and `MAESTRO_ORQ_URL` does
the same from the environment: the flag first, then the variable, then the
server kept.

The CLI keeps the server with the key and never sends the key to another: a
command pointed at a server the key wasn't made by is told to sign in there.

## Signing out

```sh
maestro logout
```

forgets the key, the server and every forge token, on this computer only.
The key and the tokens still work until you revoke them: the key on the
Maestro page, each token on its forge.
