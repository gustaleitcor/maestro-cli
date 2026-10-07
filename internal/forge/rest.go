package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"maestro-cli/internal/netfail"
)

type restAPI struct {
	base   *url.URL
	header string
	value  string
}

func (a restAPI) get(ctx context.Context, path string, query url.Values, out any) (http.Header, error) {
	target := a.base.String() + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	if a.value != "" {
		req.Header.Set(a.header, a.value)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, unreachableError(a.base.Host, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		json.Unmarshal(raw, &body)
		detail := body.Message
		if detail == "" {
			detail = body.Error
		}
		return nil, statusError(a.base.Host, resp.StatusCode, detail)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, fmt.Errorf("decoding response from %s: %w", a.base.Host, err)
	}
	return resp.Header, nil
}

// statusError turns a forge's refusal into what went wrong and what to do
// about it. detail is the forge's own message, if it gave one.
func statusError(host string, status int, detail string) error {
	said := ""
	if detail != "" {
		said = " (" + detail + ")"
	}
	switch status {
	case http.StatusUnauthorized:
		return fmt.Errorf("%s rejected the token: it is wrong, expired or revoked%s\n\nRun `maestro forges add` to store a new one", host, said)
	case http.StatusForbidden:
		return fmt.Errorf("%s does not let this token do that%s\n\nCheck the token's permissions, or run `maestro forges add` to store another", host, said)
	case http.StatusNotFound:
		return fmt.Errorf("%w on %s: it doesn't exist, or the token can't see it%s", ErrNotFound, host, said)
	case http.StatusTooManyRequests:
		return fmt.Errorf("%s is rate limiting this token; try again in a while%s", host, said)
	}
	return fmt.Errorf("%s responded with %d %s%s", host, status, http.StatusText(status), said)
}

func unreachableError(host string, err error) error {
	return netfail.Explain(host, err)
}
