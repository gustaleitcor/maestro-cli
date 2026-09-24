package maestroapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultBaseURL = "https://maestro.logsad.com"

func BaseURL() string {
	if u := os.Getenv("MAESTRO_ORQ_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return defaultBaseURL
}

type verifyKeyResponse struct {
	Valid bool   `json:"valid"`
	Email string `json:"email"`
}

func VerifyKey(ctx context.Context, key string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, BaseURL()+"/api/keys/verify", nil)
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling maestro-orq: %w", err)
	}
	defer resp.Body.Close()

	var result verifyKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	if !result.Valid {
		return "", fmt.Errorf("invalid maestro key")
	}
	return result.Email, nil
}

// BuildRequest describes what to build. Server is optional; empty means let
// maestro-orq pick.
type BuildRequest struct {
	Owner, Repo, Ref, Server string
}

// StreamEvent is one line of the /api/builds NDJSON response: either a
// stream chunk, or (only on the final line) a status.
type StreamEvent struct {
	Stream  string `json:"stream,omitempty"`
	Status  string `json:"status,omitempty"`
	ImageID string `json:"image_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// TriggerBuild POSTs to /api/builds and invokes onLine for each streamed
// output chunk as it arrives. It blocks until the build finishes or ctx is
// cancelled, returning the final status event.
func TriggerBuild(ctx context.Context, maestroKey, githubToken string, req BuildRequest, onLine func(string)) (*StreamEvent, error) {
	body, err := json.Marshal(struct {
		Owner  string `json:"owner"`
		Repo   string `json:"repo"`
		Ref    string `json:"ref"`
		Server string `json:"server,omitempty"`
	}{req.Owner, req.Repo, req.Ref, req.Server})
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, BaseURL()+"/api/builds", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+maestroKey)
	httpReq.Header.Set("X-GitHub-Token", githubToken)
	httpReq.Header.Set("Content-Type", "application/json")

	// No client timeout: builds can run for many minutes. Cancel via ctx instead.
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling maestro-orq: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error == "" {
			return nil, fmt.Errorf("maestro-orq responded with %s", resp.Status)
		}
		return nil, fmt.Errorf("maestro-orq: %s", errResp.Error)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024) // build log lines can be long

	var final StreamEvent
	for scanner.Scan() {
		var evt StreamEvent
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			continue
		}
		if evt.Status != "" {
			final = evt
			continue
		}
		onLine(evt.Stream)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading build stream: %w", err)
	}
	if final.Status == "" {
		return nil, fmt.Errorf("build stream ended unexpectedly without a final status")
	}
	return &final, nil
}

type Image struct {
	ID        string    `json:"id"`
	Tag       string    `json:"tag"`
	BuildID   int64     `json:"build_id"`
	Repo      string    `json:"repo"`
	Ref       string    `json:"ref"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// ListImages returns the images built for the key's owner, newest first.
func ListImages(ctx context.Context, maestroKey string) ([]Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, BaseURL()+"/api/images", nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+maestroKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling maestro-orq: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error == "" {
			return nil, fmt.Errorf("maestro-orq responded with %s", resp.Status)
		}
		return nil, fmt.Errorf("maestro-orq: %s", errResp.Error)
	}

	var images []Image
	if err := json.NewDecoder(resp.Body).Decode(&images); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return images, nil
}
