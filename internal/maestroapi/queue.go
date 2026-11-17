package maestroapi

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// Queue counts lines in each status, queued, assigned, starting and running:
// everyone's in Totals, and in Users the caller's, or every user's for an
// administrator. Heads is how many of a user's queued lines are in the
// dispatcher queue, 0 or 1. Share is a user's share of the cluster over
// everyone's, from 0 to 1, as the scheduler weighs it. Groups are how many
// of each group's lines are queued, how many of those are heads and how
// many are assigned: the caller's group, or every group for an
// administrator. A server from before groups sends none, and one from
// before the three queues counts staged lines too, with no heads.
type Queue struct {
	Totals map[string]int64 `json:"totals"`
	Users  []struct {
		ID    int64            `json:"id"`
		Name  string           `json:"name"`
		Email string           `json:"email"`
		Group int64            `json:"group_id"`
		Lines map[string]int64 `json:"lines"`
		Heads int64            `json:"heads"`
		Share float64          `json:"share"`
		// AtCeiling says their head waits because of UserMaxCPUs.
		AtCeiling bool `json:"at_ceiling"`
	} `json:"users"`
	Groups    []QueueGroup `json:"groups"`
	Scheduler Scheduler    `json:"scheduler"`
	// UserMaxCPUs is the most cores a user's lines assigned, starting and
	// running may take together; 0 means no ceiling. AtCeiling says the
	// caller's head waits because of it.
	UserMaxCPUs int64 `json:"user_max_cpus"`
	AtCeiling   bool  `json:"at_ceiling"`
}

// QueueGroup is one group's waiting lines. Staged and Capacity are only
// from a server from before the three queues: its staged lines, out of
// how many it could have.
type QueueGroup struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Queued   int64  `json:"queued"`
	Heads    int64  `json:"heads"`
	Assigned int64  `json:"assigned"`
	Staged   int64  `json:"staged,omitempty"`
	Capacity int    `json:"capacity,omitempty"`
}

// Usage is what one user's lines used each day, UTC, counted as they end.
type Usage struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Days  []struct {
		Day           string  `json:"day"`
		CPUHours      float64 `json:"cpu_hours"`
		MemoryGBHours float64 `json:"memory_gb_hours"`
	} `json:"days"`
}

func GetQueue(ctx context.Context, maestroKey string) (*Queue, error) {
	var q Queue
	if err := doWithKey(ctx, maestroKey, http.MethodGet, "/api/queue", nil, http.StatusOK, &q); err != nil {
		return nil, err
	}
	return &q, nil
}

// GetUsage is the last days of usage: the caller's, or everyone's for an
// administrator; user, when not 0, picks one user.
func GetUsage(ctx context.Context, maestroKey string, days int, user int64) ([]Usage, error) {
	query := url.Values{"days": {strconv.Itoa(days)}}
	if user != 0 {
		query.Set("user", strconv.FormatInt(user, 10))
	}
	var usage []Usage
	if err := doWithKey(ctx, maestroKey, http.MethodGet, "/api/usage?"+query.Encode(), nil, http.StatusOK, &usage); err != nil {
		return nil, err
	}
	return usage, nil
}

// ExportLines writes to w, as CSV, what happened to each line of the runs
// queued from since until before until (both a day, 2026-11-01, or an RFC
// 3339 time; until may be empty for now): the caller's, or with all
// everyone's, for an administrator.
func ExportLines(ctx context.Context, maestroKey, since, until string, all bool, w io.Writer) error {
	query := url.Values{"since": {since}}
	if until != "" {
		query.Set("until", until)
	}
	if all {
		query.Set("all", "true")
	}
	return download(ctx, maestroKey, "/api/lines/export?"+query.Encode(), w)
}
