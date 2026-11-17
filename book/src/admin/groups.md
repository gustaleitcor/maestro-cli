# Groups

Every user is in one group, and groups take turns at the machines. Each
group has a dispatcher queue of its own, made of its users' heads, so a
group with many users can't crowd out a group with few.

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

Moving a user affects the lines assigned from then on. Their queued lines
wait in the new group's dispatcher queue. Lines already assigned keep the
group they were assigned in.

## How groups affect the queue

Inside a group, the `dispatcher` policy chooses whose head goes to a
machine next. Between groups no policy chooses. The groups take turns, one
line a turn.
[The queue](../how/queue.md#the-dispatcher-queue-from-queued-to-assigned)
describes a turn.

`maestro queue` shows each user their group: how many of its lines are
queued, how many of those are heads in its dispatcher queue, and how many
are assigned.

Each line records the group its user was in when the line was assigned.
[`maestro export`](../use/usage.md#maestro-export) writes it, so moving a
user later doesn't change what the export says about earlier lines.
