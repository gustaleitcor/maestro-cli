# Quickstart: run the template

This page takes you from nothing to a finished run and its results. It uses
[maestro-template](https://github.com/gustaleitcor/maestro-template), a small
public repository made for this, so you don't need a repository of your own
yet.

## 1. Install the CLI

Download the package for your system from the
[maestro-cli releases](https://github.com/gustaleitcor/maestro-cli/releases)
and install it:

```sh
sudo dnf install ./maestro-*.x86_64.rpm     # Fedora, RHEL, Rocky, Alma
sudo apt install ./maestro_*_amd64.deb      # Debian, Ubuntu, Mint
```

[Installing the CLI](../use/install.md) has the command for other systems,
and for installing without root.

## 2. Sign in

```sh
maestro login
```

The first time, `maestro` asks for the address of your Maestro server and
keeps it:

```text
No Maestro server is set yet.
Maestro server URL (e.g. https://maestro.example.org): maestro.example.org
Kept. `maestro login --server <url>` changes it.
```

Then it prints a link and a short code:

```text
To sign in to Maestro, open:

  https://maestro.example.org/?cli=KQ7M-W2HX

and approve this code:

  KQ7M-W2HX

Waiting for approval...
```

Open the link in any browser, sign in with your lab account if the page asks,
and approve the code. The browser can be on another computer, which helps
when you reach this one over SSH. The terminal then confirms:

```text
Maestro: logged in to https://maestro.example.org as ada@example.com.
```

`maestro login` ends by offering to add a git forge. Answer `n` for now. The
template is public, and Maestro needs a forge only to read private
repositories.

## 3. Build and run the template

```sh
maestro build https://github.com/gustaleitcor/maestro-template --watch
```

This command clones the repository on the server, builds a container image
from its Dockerfile, starts a run of that image, and follows the run until it
is over. The build log streams to your terminal. These are the lines to look
for in it:

```text
Building github.com/gustaleitcor/maestro-template@main...
Cloning github.com/gustaleitcor/maestro-template@main...
Read maestro.toml: 2 container(s) on any machine, with 1 CPU and 512 MB each, keeping out.txt.
STEP 1/7: FROM docker.io/library/python:3.12-slim
…
Build #12 succeeded: 5d0c3a9f2e41…
Queued run 31 of build #12: 2 line(s) on any machine.
Keeping out.txt from each container.
```

The line that starts with `Read maestro.toml` says what a run of this build
will do. The template's `maestro.toml` lists two sets of parameters, so the
run has two containers. Maestro calls each one a line.

After the build, `--watch` draws the run and redraws it every two seconds:

```text
Run 31 of gustaleitcor/maestro-template@main (build #12): running, on any machine
2 line(s): 1 assigned, 1 running

LINE  PARAMETERS                        MACHINE  STATUS    FILES
1     -t 10 out.txt                     Q1       running   -
2     -t 20 --label 'long run' out.txt  Q2       assigned  -
```

The numbers and machine names will be your server's. A line's status goes
from `queued` to `assigned` while it waits, then `running`, then
`succeeded`. [The queue](../how/queue.md) explains the waiting ones. The
template's program sleeps for the number of seconds after `-t`, so the run
takes under a minute once its lines start. The command returns when both
lines have ended:

```text
Run 31 of gustaleitcor/maestro-template@main (build #12): succeeded, on any machine
2 line(s): 2 succeeded

LINE  PARAMETERS                        MACHINE  STATUS     FILES
1     -t 10 out.txt                     Q1       succeeded  out.txt
2     -t 20 --label 'long run' out.txt  Q2       succeeded  out.txt
```

## 4. Look at what the run left

Each line keeps what its container printed and the files `maestro.toml`
names, here `out.txt`. Use the run number from the previous step.

```sh
maestro runs logs 31 2     # what line 2 printed
```

```text
long run: waiting 20s
done
```

```sh
maestro runs get 31        # download everything into ./31/
```

```text
Downloaded 2 file(s) and the logs into 31
```

```text
31/
├── 1/
│   ├── out.txt
│   ├── stderr.log
│   └── stdout.log
└── 2/
    ├── out.txt
    ├── stderr.log
    └── stdout.log
```

`31/2/out.txt` holds `long run: done after 20s`, the result the program wrote.

The same files are in the browser. Open your Maestro server's address, and
the page shows your runs. Click run 31, then a line, and its files open
beside it.

## What happened

1. Maestro cloned the repository and built an image from its Dockerfile.
   That is build #12.
2. It read `maestro.toml`, which lists two lines of parameters and one file
   to keep.
3. `--watch` started run 31 of that build, with one container for each line.
4. Each container ran the image's program with its line as arguments, on a
   machine that had a free core and 512 MB.
5. When a container ended, Maestro kept its output and `out.txt`, then
   removed the container.

A build and a run have separate numbers. `maestro run 12` starts another run
of the same image without building again, and `maestro runs` lists the runs
you have.

## Next

[Your own copy of the template](your-copy.md) gives you a repository to
change. The chapters after it use that copy to explain the Dockerfile,
`maestro.toml` and the generator.
