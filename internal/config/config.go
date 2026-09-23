// Package config handles loading, saving and locating the Maestro CLI's
// local configuration: the GitHub token and Maestro key used to authenticate
// API requests.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	service = "maestro-cli"

	githubKeyringUser  = "github_token"
	maestroKeyringUser = "maestro_key"

	appDirName  = "maestro"
	configFile  = "config.json"
	fileEnvFlag = "MAESTRO_FORCE_FILE_STORE" // set to "1" to skip keyring and force file storage
)

// Config is the persisted CLI configuration.
//
// Both fields are only ever populated here as a fallback: when the OS
// keyring is unavailable, the corresponding secret is stored in this file
// instead (with restrictive permissions). When the keyring is available,
// these fields are left empty on disk and the secrets live only in the
// keyring.
type Config struct {
	GitHubToken string `json:"github_token,omitempty"`
	MaestroKey  string `json:"maestro_key,omitempty"`
}

func dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolving user config dir: %w", err)
	}
	return filepath.Join(base, appDirName), nil
}

func path() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, configFile), nil
}

func ensureDir() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return "", fmt.Errorf("creating config dir %s: %w", d, err)
	}
	return d, nil
}

// ErrNoGitHubToken is returned when no GitHub token is found in either the
// keyring or the fallback file.
var ErrNoGitHubToken = errors.New("no github token found; run `maestro login` first")

// ErrNoMaestroKey is returned when no Maestro key is found in either the
// keyring or the fallback file.
var ErrNoMaestroKey = errors.New("no maestro key found; run `maestro login` first")

// SaveGitHubToken persists the GitHub token, preferring the OS keyring and
// falling back to a 0600 file in the config directory when no keyring
// backend is available (e.g. headless Linux without a Secret Service).
func SaveGitHubToken(token string) error {
	return saveSecret(githubKeyringUser, token, func(c *Config) *string { return &c.GitHubToken })
}

// LoadGitHubToken retrieves the GitHub token, checking the OS keyring first
// and falling back to the config file.
func LoadGitHubToken() (string, error) {
	return loadSecret(githubKeyringUser, ErrNoGitHubToken, func(c *Config) string { return c.GitHubToken })
}

// DeleteGitHubToken removes the stored GitHub token from both the keyring
// and the fallback file, ignoring "not found" errors from either.
func DeleteGitHubToken() error {
	return deleteSecret(githubKeyringUser, func(c *Config) *string { return &c.GitHubToken })
}

// SaveMaestroKey persists the Maestro key, with the same keyring-first,
// file-fallback behavior as SaveGitHubToken.
func SaveMaestroKey(key string) error {
	return saveSecret(maestroKeyringUser, key, func(c *Config) *string { return &c.MaestroKey })
}

// LoadMaestroKey retrieves the Maestro key, checking the OS keyring first
// and falling back to the config file.
func LoadMaestroKey() (string, error) {
	return loadSecret(maestroKeyringUser, ErrNoMaestroKey, func(c *Config) string { return c.MaestroKey })
}

// DeleteMaestroKey removes the stored Maestro key from both the keyring and
// the fallback file, ignoring "not found" errors from either.
func DeleteMaestroKey() error {
	return deleteSecret(maestroKeyringUser, func(c *Config) *string { return &c.MaestroKey })
}

func saveSecret(keyringUser, value string, field func(*Config) *string) error {
	if os.Getenv(fileEnvFlag) != "1" {
		if err := keyring.Set(service, keyringUser, value); err == nil {
			// Clear any stale value from an earlier file-based save so it doesn't linger on disk.
			return clearFileField(field)
		}
		// Keyring unavailable or failed — fall through to file storage.
	}
	return saveFieldToFile(field, value)
}

func loadSecret(keyringUser string, notFoundErr error, field func(*Config) string) (string, error) {
	if os.Getenv(fileEnvFlag) != "1" {
		value, err := keyring.Get(service, keyringUser)
		if err == nil {
			return value, nil
		}
		// err is either ErrNotFound or a backend failure — fall back to the file either way.
	}
	return loadFieldFromFile(notFoundErr, field)
}

func deleteSecret(keyringUser string, field func(*Config) *string) error {
	_ = keyring.Delete(service, keyringUser)
	return clearFileField(field)
}

// saveFieldToFile and clearFileField read-modify-write the config file so
// saving or clearing one secret never clobbers the other.

func saveFieldToFile(field func(*Config) *string, value string) error {
	if _, err := ensureDir(); err != nil {
		return err
	}
	cfg, err := readConfigFile()
	if err != nil {
		return err
	}
	*field(&cfg) = value
	return writeConfigFile(cfg)
}

func clearFileField(field func(*Config) *string) error {
	p, err := path()
	if err != nil {
		return err
	}
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}
	cfg, err := readConfigFile()
	if err != nil {
		return err
	}
	*field(&cfg) = ""
	return writeConfigFile(cfg)
}

func loadFieldFromFile(notFoundErr error, field func(*Config) string) (string, error) {
	cfg, err := readConfigFile()
	if err != nil {
		return "", err
	}
	if value := field(&cfg); value != "" {
		return value, nil
	}
	return "", notFoundErr
}

// readConfigFile returns a zero-value Config, not an error, when the file
// doesn't exist yet — the natural state before the first `maestro login`.
func readConfigFile() (Config, error) {
	p, err := path()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("reading config file %s: %w", p, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file %s: %w", p, err)
	}
	return cfg, nil
}

func writeConfigFile(cfg Config) error {
	p, err := path()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	// Write with 0600 from the start. Note os.WriteFile only applies the
	// mode on creation, so if the file already existed with looser
	// permissions we chmod it explicitly afterward.
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return fmt.Errorf("writing config file %s: %w", p, err)
	}
	if err := os.Chmod(p, 0o600); err != nil {
		return fmt.Errorf("setting permissions on %s: %w", p, err)
	}
	return nil
}
