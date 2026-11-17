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

A key expires 90 days after it was issued, unless your administrator set
another length. After that, commands say "this Maestro key expired; run
`maestro login`", and `maestro login` gives you a new one.

The CLI doesn't open the link for you. You can open it in a browser on
another computer, which is how you sign in on a computer you reach over SSH.

## Where the key is kept

The CLI stores the key in your system's keyring. On a computer without a
keyring, or one whose keyring asks for a password each time, sign in with:

```sh
maestro login --store-in-file
```

The key then goes into the CLI's config file, which only your user can read.

## Which server the CLI talks to

The CLI has no built-in server. The first time a command needs one, `maestro`
asks for its address and keeps it. An address without a scheme gets
`https://`. A script has no terminal to answer at, so there the command
stops and tells you to pass `--server <url>`, set `MAESTRO_ORQ_URL`, or run
`maestro login --server <url>` once.

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

## Your keys

Each `maestro login` issues a new Maestro key, for the computer you run it
on. To see the keys you have:

```sh
maestro keys
```

It lists each key by its prefix, the first 10 characters, with when it was
created and when it expires. It marks the key this computer uses.

If you no longer trust a computer you signed in on, revoke its key from
another one:

```sh
maestro keys rm mk_Q3xv9aZ
```

The key stops working at once, on every computer that holds it. The command
asks first. With --yes it doesn't ask.

An administrator can revoke every key of a user, from Admin › Groups on the
Maestro page.

## Signing out

```sh
maestro logout
```

This revokes the key on the server that issued it. Then it removes the key,
the server address and every forge token from this computer.

If the server can't be reached, `maestro logout` still forgets the key, and
says that the key is still valid on the server. Revoke it later with
`maestro keys rm` from a computer that is signed in.

`maestro logout` changes nothing on the forges, so revoke a token there if
you no longer trust the computer it was on.
