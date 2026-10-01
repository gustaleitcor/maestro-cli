package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// restAPI is the plain JSON-over-HTTP client the Forgejo and GitLab
// implementations share; they differ only in paths and the auth header.
type restAPI struct {
	base   *url.URL
	header string
	value  string
}

// get decodes the JSON at path (which must already be escaped) into out and
// returns the response headers, which carry the pagination.
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
		return nil, fmt.Errorf("calling %s: %w", a.base.Host, err)
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
		if detail == "" {
			return nil, fmt.Errorf("%s responded with %s", a.base.Host, resp.Status)
		}
		return nil, fmt.Errorf("%s responded with %s: %s", a.base.Host, resp.Status, detail)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, fmt.Errorf("decoding response from %s: %w", a.base.Host, err)
	}
	return resp.Header, nil
}
