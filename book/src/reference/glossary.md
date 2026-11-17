# Glossary

`maestro help concepts` prints a short version of this page in the
terminal.

## The things you work with

| Word | Meaning | More |
| --- | --- | --- |
| Repo | A git repository of yours on GitHub, GitLab, Codeberg, or another Forgejo or Gitea server. It holds your program and a Dockerfile, and usually a `maestro.toml` that says how to run the image. | [Your own copy of the template](../start/your-copy.md) |
| Forge | The place a repository lives, as Maestro reaches it: the forge's API and a read-only token of yours. The CLI keeps the token and sends it to the server when the server needs to clone. A forge never signs anyone in. | [Git forges](../use/forges.md) |
| Build | The result of `maestro build <repo>`, which clones a repository at one commit and builds a container image from it. A build has a number, written `#12`. The build also reads `maestro.toml` and computes the lines, so every run of a build has the same lines. | [Builds and images](../use/builds.md) |
| Image | What a build makes: your program and everything it needs, ready to start as a container. You keep 10 at a time unless your administrator set another number. When a new build needs room, the image you ran least recently is removed. | [How many images you keep](../use/builds.md#how-many-images-you-keep) |
| Run | The result of `maestro run <build>`, which runs a build's image on the machines. A run has a number of its own, written `run 31`. Build numbers and run numbers count separately. | [Runs and lines](../use/runs.md) |
| Line | One container of a run. `maestro.toml` lists lines of parameters and can name a generator that computes more. A run starts one container for each line, with the line's words in place of the image's `CMD`. Without a `maestro.toml`, a run is one line that runs the image's own command. A run has at most 5000 lines. | [`parameters`](../repo/maestro-toml.md#parameters) |
| Generator | A Python script in the repository whose `genvrs` function yields lines. | [Generating lines with Python](../repo/generator.md) |
| Outputs | The files `maestro.toml` names, copied out of each container when it ends. They are kept beside what the container printed, `stdout.log` and `stderr.log`. `maestro runs get` downloads all of it. | [What a line leaves behind](../use/results.md) |
| Machine | A computer whose Podman runs the containers. Maestro reaches it over SSH. A line runs on one machine, the first allowed one that has room for its cores and memory. | [Watching the machines](../use/machines.md) |
| Maestro key | What the CLI signs in with. `maestro login` has the server issue one for your computer, and the CLI sends it with every command. A key is shown by its prefix, its first 10 characters, such as `mk_Q3xv9aZ`. It expires 90 days after it was issued, unless your administrator set another length. `maestro keys rm` revokes one, `maestro logout` revokes the one it holds, and an administrator can revoke all of a user's. A revoked key stops working at once. | [Signing in](../use/login.md#your-keys) |
| Group | A set of users. Groups take turns at the machines, so a lab of ten doesn't crowd out a lab of two. Everyone is in one group until an administrator makes more. | [Groups](../admin/groups.md) |

## The statuses of a line

A line waits in three queues before it runs: its user's queue, its group's
dispatcher queue as its user's head, and a machine's queue.

| Status | The line is |
| --- | --- |
| `queued` | In its user's queue. The first of a user's queued lines is their head, which waits in the dispatcher queue for a machine. |
| `assigned` | Given to a machine, and waiting for room on it while its image is copied there |
| `starting`, `running` | In a container on the machine |

It then ends in one of four ways.

| Status | What happened |
| --- | --- |
| `succeeded` | The program exited with 0. |
| `failed` | The program exited with another code. |
| `error` | The line couldn't run, for a reason on Maestro's side and outside your program. A line that runs longer than the server allows is stopped and ends as `error` too. |
| `cancelled` | Its run was cancelled before it ended. |

[The queue](../how/queue.md) explains the three queues, and
[How a line ends](../use/runs.md#how-a-line-ends) the endings.

## The programs

| Name | What it is |
| --- | --- |
| orq | The server. It builds images, schedules lines and keeps every record. |
| view | The Maestro page in your browser |
| CLI | The `maestro` command |

[The parts](../how/parts.md) shows how they connect.

## The scheduler's words

| Word | Meaning |
| --- | --- |
| User queue | A user's queued lines, in the order the `user` policy gives them |
| Head | The first line of a user's queue, the one their group's dispatcher queue holds for them |
| Dispatcher queue | The heads of a group's users. In the group's turn, the `dispatcher` policy gives one of them a machine. |
| Machine queue | The lines assigned to a machine that wait for room on it, at most `depth` of them |
| Policy | The rule that picks who leaves one of the three queues, such as `fifo` or `pq` |
| Score | The number `pq` ranks lines by, built from a line's cores, memory and waiting time and its user's share |
| Share | How much of the machines a user holds or has used, as the chosen measure counts it |

[The queue](../how/queue.md) describes the queues, and [Policies and the score](../how/score.md) the policies, the score and the share.
