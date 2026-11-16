# Git forges

A forge is the place your repositories live, such as GitHub or Codeberg.
Maestro needs a forge only to read private repositories, and to let you name
a repository without typing its whole address. A public repository on
GitHub, GitLab or Codeberg builds from its address with no forge at all.

To Maestro, a forge is the address of its API and a read-only token of
yours. The CLI keeps the token on your computer and sends it to the server
with each build that needs it. Maestro uses it to list and clone
repositories. A forge never signs you in to Maestro.

```sh
maestro forges add          # add one: asks for the kind and the token
maestro forges              # the forges you have
maestro repos               # the repositories they can see
maestro forges rm <name>    # forget one
```

## The kinds of forge

| Kind | Where | Token to create |
| --- | --- | --- |
| `github` | github.com | A classic token with the `repo` scope |
| `forgejo` | Codeberg, or any Forgejo or Gitea server with `--url` | Repository: Read and User: Read, everything else at No access |
| `gitlab` | gitlab.com, nested groups included | The `read_api` and `read_repository` scopes |

`maestro forges add` tells you how to create the token. For GitHub and
GitLab it prints a link that fills in the form. It checks the token before
keeping it.

On GitHub, use a classic token. A fine-grained token reaches the
repositories of one owner only, so it can't read a private repository that
belongs to someone else and that you collaborate on.

GitHub and GitLab tokens expire after 30 days unless you choose otherwise
when you create them. When one expires, run `maestro forges add` with the
same name to replace it.

## Naming a repository

`maestro build` accepts a name, an `owner/name`, or an address:

```sh
maestro build my-app                                 # yours, on your forge
maestro build some-org/my-app --ref v1.2.0           # someone else's, at a tag
maestro build https://codeberg.org/some-org/my-app   # by address
```

Maestro uses the forge that has the repository. When more than one has it,
choose with `--forge`.

## Your lab's own Forgejo

Add it with its address:

```sh
maestro forges add --kind forgejo --url https://git.example.org
```

The server clones only from hosts its administrator allows. By default
those are `github.com`, `codeberg.org` and `gitlab.com`, so an administrator
has to add your lab's host to `ALLOWED_FORGE_HOSTS` first
([Environment variables](../reference/env.md)).

## Browsing repositories

In a terminal, `maestro repos` is a full-screen list of one forge's
repositories. The left and right arrows switch to another forge, and
`--forge` chooses the one shown first. With `--json`, or when its output
goes to a pipe, it prints every forge's repositories once.
