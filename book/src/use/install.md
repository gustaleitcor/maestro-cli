# Installing the CLI

The CLI is one program, `maestro`. Every release has packages for Linux and a
plain binary for Linux, macOS and Windows on the
[maestro-cli releases](https://github.com/gustaleitcor/maestro-cli/releases)
page.

## With a package

A package installs `maestro` in `/usr/bin`, with its shell completions:

```sh
sudo dnf install ./maestro-*.x86_64.rpm                            # Fedora, RHEL, Rocky, Alma
sudo zypper install --allow-unsigned-rpm ./maestro-*.x86_64.rpm    # openSUSE
sudo apt install ./maestro_*_amd64.deb                             # Ubuntu, Debian, Mint, Pop!_OS
sudo apk add --allow-untrusted ./maestro_*_x86_64.apk              # Alpine
sudo pacman -U ./maestro-*-x86_64.pkg.tar.zst                      # Arch, Manjaro, EndeavourOS
```

On an ARM machine, take the `aarch64` or `arm64` package.

## Without root, or on macOS and Windows

The `.tar.gz` archive (`.zip` on Windows) holds the same binary, which needs
nothing else installed. Unpack it and put `maestro` in a directory on your
`PATH`, such as `~/.local/bin`.

## Shell completions

The packages install completions for bash, fish and zsh. With the plain
binary, write them yourself:

```sh
maestro completion bash > ~/.local/share/bash-completion/completions/maestro
maestro completion fish > ~/.config/fish/completions/maestro.fish
maestro completion zsh  > "${fpath[1]}/_maestro"
```

Completions offer your own builds, runs, machines and repositories.
`maestro run <TAB>` lists your builds, each with the repository it was built
from. They ask the server, so they work once you have
[signed in](login.md).

## Checking the install

```sh
maestro --version
maestro docs
```

`maestro docs` opens this book from the copy inside the binary, which is a
quick way to confirm the binary on your `PATH` is the one you installed.
