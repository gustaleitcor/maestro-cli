# Getting started

Four steps take you from nothing to a run's results: install the CLI, [sign
in](login.md), [connect a forge](forges.md), and [run something](first-run.md).
The Maestro page, in a browser, needs only the second.

## Installing the CLI

Install `maestro` from the
[maestro-cli releases](https://github.com/gustaleitcor/maestro-cli/releases).
On Linux, a package installs it in `/usr/bin` with its shell completions:

```sh
sudo dnf install ./maestro-*.x86_64.rpm            # Fedora, RHEL, Rocky, Alma
sudo zypper install --allow-unsigned-rpm ./maestro-*.x86_64.rpm   # openSUSE
sudo apt install ./maestro_*_amd64.deb             # Ubuntu, Debian, Mint, Pop!_OS
sudo apk add --allow-untrusted ./maestro_*_x86_64.apk             # Alpine
sudo pacman -U ./maestro-*-x86_64.pkg.tar.zst      # Arch, Manjaro, EndeavourOS
```

On ARM machines, take the aarch64 or arm64 one. Elsewhere, or without root,
the `.tar.gz` (or, on Windows, the `.zip`) holds the same binary, which needs
nothing else: put it anywhere on your `PATH`.

## Shell completions

The packages install completions for bash, fish and zsh. With the binary
alone, have your shell load them:

```sh
maestro completion bash > ~/.local/share/bash-completion/completions/maestro
maestro completion fish > ~/.config/fish/completions/maestro.fish
maestro completion zsh  > "${fpath[1]}/_maestro"
```

Completions know your builds, runs, machines and repos, so `maestro run <TAB>`
offers your builds with the repo each was built from. They ask the server,
so they need you signed in.

## How the CLI talks

`maestro --help` lists the commands by what they are for, and every
command's `--help` ends with the page of this book that explains it.

What there is several of has a command in the plural, which lists when given
nothing else: `maestro builds`, `runs`, `machines`, `repos` and `forges` are
their `list`. Each list takes `--json`, which prints what the table shows, as
the server names it, for scripts. Removing is `rm` everywhere (`builds rm`,
`runs rm`, `forges rm`), which asks first; without a terminal to ask at, as
in a script, it takes `--yes`. The names from before still work: `image`,
`machine`, `repo`, `forge`, and `remove` for `rm`.

For scripts, `maestro build` and `maestro run` take `--json`, which prints
what they made, or `-q`, which prints only its number; either keeps stdout
for that, and what they say along the way goes to stderr:

```sh
run=$(maestro run 12 -q) && maestro runs show "$run" --watch
```

`maestro` exits 0 when all went well; 1 when something failed, a run watched
to its end with `--watch` that failed or was cancelled included; 2 when a
command was asked wrongly, which it says before asking the server anything;
and 130 when ctrl-c stopped it first. `--no-color` (or `NO_COLOR=1`) prints
without colours.
