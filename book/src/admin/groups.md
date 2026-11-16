# Groups

Every user is in one group, and groups take turns at the machines. Each
group's lines are staged separately, a limited number at a time, so a group
with many users can't crowd out a group with few.

A new Maestro has one group, the default group, and everyone is in it. New
users start there. It stays the only group until an administrator makes
more.

## Managing groups

Admin › Groups on the Maestro page lists the groups and their members.

- "New group" creates a group, and Rename changes a group's name.
- "Add member" moves a user into a group. A user is in one group at a time,
  so this takes the user out of the group they were in.
- The ✕ beside a member sends that user back to the default group.
- Remove deletes a group and sends its users back to the default group. The
  default group itself can't be removed.

Moving a user affects the lines staged from then on. Lines already staged
keep the group they were staged in.

## How groups affect the queue

Inside a group, the `select` policy chooses whose line is staged next.
Between groups no policy chooses. The groups take turns, one line a turn. [The queue](../how/queue.md#allocation-from-staged-to-assigned)
describes a turn.

`maestro queue` shows each user their group, how many of its lines are
staged, and how many it may have staged at once.

Each line records the group its user was in when the line was staged.
[`maestro export`](../use/usage.md#maestro-export) writes it, so moving a
user later doesn't change what the export says about earlier lines.
