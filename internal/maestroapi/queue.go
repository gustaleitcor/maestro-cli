package maestroapi

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// Queue counts lines at each level, queued, staged, assigned, starting and
// running: everyone's in Totals, and in Users the caller's, or every user's
// for an administrator. Share is a user's share of the cluster over
// everyone's, from 0 to 1, as the scheduler weighs it.
type Queue struct {
	Totals map[string]int64 `json:"totals"`
	Users  []struct {
		ID    int64            `json:"id"`
		Name  string           `json:"name"`
		Email string           `json:"email"`
		Lines map[string]int64 `json:"lines"`
		Share float64          `json:"share"`
	} `json:"users"`
	Scheduler Scheduler `json:"scheduler"`
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
