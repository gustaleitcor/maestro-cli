package maestroapi

import (
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
