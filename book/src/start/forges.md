# Git forges

To build a private repo, Maestro needs to read it. A forge is where repos
live, as Maestro reaches it: the forge's API and a token of yours. The CLI
keeps the token and hands it to the server with each build that needs it;
Maestro only ever reads with it. Forges are separate from signing in: they
never log anyone in.

```sh
maestro forges add          # asks which kind, and for the token
maestro forges              # the forges you have
maestro repos               # every repo they can see
maestro forges rm codeberg  # forget one
```

`maestro login` offers to add one when you have none.

## Kinds of forge

| Kind | Where | Token |
| --- | --- | --- |
| `github` | github.com | a *classic* token with the `repo` scope |
| `forgejo` | Codeberg by default, or any Forgejo or Gitea instance with `--url` | repository: Read and user: Read, nothing else |
| `gitlab` | gitlab.com, nested groups included | the `read_api` and `read_repository` scopes |

`maestro forges add` says how to make the token, with a link that fills in
the form where the forge allows it, and checks the token before keeping it.
A fine-grained GitHub token only reaches the repos of the one owner it is
made for, so never another user's private repo you are a collaborator on:
use a classic one. GitHub and GitLab tokens expire in 30 days by default;
`maestro forges add` with the same name replaces an expired one.

## Naming a repo

`maestro build` takes a repo as a name, `owner/name`, or a URL. The forge
that has it is used; pass `--forge` when several do:

```sh
maestro build my-app
maestro build some-org/my-app --ref v1.2.0
maestro build https://codeberg.org/some-org/my-app
```

A public repo on GitHub, GitLab or Codeberg needs no forge and no token:
pass its full URL.

## Self-hosted forges

The server only clones from the hosts its administrator allows
(`ALLOWED_FORGE_HOSTS`, by default `github.com`, `codeberg.org` and
`gitlab.com`), so that it can't be pointed at hosts on its own network. A
lab's own Forgejo or GitLab has to be added there first; see [Environment
variables](../reference/env.md).

## Full screen

On a terminal, `maestro repos` is a full-screen list of one forge's
repositories, and the left and right arrows switch forge (`--forge` picks
the first). Piped somewhere, or with `--json`, it prints every forge's
repositories once.
