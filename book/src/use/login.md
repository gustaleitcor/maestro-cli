# Signing in

Maestro has no passwords of its own. It signs you in through your lab's
identity provider, with the account you use for the lab's other services.

## In the browser

Open the Maestro page and press the sign-in button. The identity provider
asks for your account, and sends you back signed in. A session lasts 12
hours unless your administrator set another length.

Signing out of Maestro ends the Maestro session and leaves you signed in at
the identity provider. Signing out there doesn't end a Maestro session that
is already open.

## In the terminal

```sh
maestro login
```

`maestro login` prints a link to the Maestro page and a short code. Open the
link, sign in there if the page asks, and approve the code. The CLI then
receives a key of its own and uses it for every command.

The CLI doesn't open the link for you. You can open it in a browser on
another computer, which is how you sign in on a machine you reach over SSH.

## Where the key is kept

The CLI stores the key in your system's keyring. On a machine without a
keyring, or one whose keyring asks for a password each time, sign in with:

```sh
maestro login --store-in-file
```

The key then goes into the CLI's config file, which only your user can read.

## Which server the CLI talks to

The CLI has no built-in server. The first time a command needs one, `maestro`
asks for its address and keeps it. An address without a scheme gets
`https://`.

To use another Maestro, sign in to it:

```sh
maestro login --server https://maestro.example.org
```

Every later command talks to that server. For one command only, pass
`--server`, or set `MAESTRO_ORQ_URL`. The flag wins over the variable, and
the variable wins over the server the CLI kept.

The CLI stores the key together with the server that issued it and sends it
nowhere else. If you point a command at a different server, the CLI tells
you to sign in there.

`maestro` with no arguments shows who is signed in, on which server, and on
which forges.

## Signing out

```sh
maestro logout
```

This removes the key, the server address and every forge token from this
computer. It changes nothing on the forges, so revoke a token there if you
no longer trust the computer it was on.
