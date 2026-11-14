# Groups

Users are in groups, and groups take turns at the machines: each group's
lines are staged apart, only a few at a time, so a group of many users
doesn't crowd out a group of few. Until an administrator makes others,
everyone is in one group, open to all.

Admin › Groups, on the Maestro page, has the groups: making, renaming and
removing them, and moving users between them (Add member, or ✕ to send
someone back to the default group). A user is in one group at a time.

Within a group, the **select** policy chooses whose line is staged next;
between groups, they take turns. See [Scheduler policies](policies.md).

`maestro queue` shows each user their group, how many of its lines are
staged, and how many it may have staged at once. Each line records the
group its user was in when it was staged, for [`maestro
export`](../watch/usage.md).
