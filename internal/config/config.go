package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	service = "maestro-cli"

	githubKeyringUser  = "github_token" // legacy: migrated to a forge named "github"
	maestroKeyringUser = "maestro_key"
	forgeKeyringPrefix = "forge:"

	storeFile = "file"

	appDirName  = "maestro"
	configFile  = "config.json"
	fileEnvFlag = "MAESTRO_FORCE_FILE_STORE" // set to "1" to skip keyring and force file storage
)

var ErrNoGitHubToken = errors.New("no github token found; run `maestro login` first")

var ErrNoMaestroKey = errors.New("not signed in to Maestro")

type Config struct {
	GitHubToken string `json:"github_token,omitempty"`
	MaestroKey  string `json:"maestro_key,omitempty"`
	// Server is the Maestro server the key was minted by, wherever the key
	// itself is kept; empty for a key stored before servers were.
	Server string `json:"server,omitempty"`

	// Store is "file" once the user chose to keep secrets in this file and
	// not in the system keyring; see UseFileStore.
	Store string `json:"store,omitempty"`

	Forges []Forge `json:"forges,omitempty"`
	// ForgeTokens is only used when the keyring isn't; see saveForgeToken.
	ForgeTokens map[string]string `json:"forge_tokens,omitempty"`
}

type Forge struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	BaseURL string `json:"base_url,omitempty"` // empty: the kind's public instance
}

// A missing file isn't an error here — returns a zero-value Config instead.
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

// useKeyring is whether secrets go to, and are looked for in, the system
// keyring. Some keyrings ask for a password every time they are touched, so
// a user can opt out of them for good with UseFileStore, or for one run with
// MAESTRO_FORCE_FILE_STORE=1.
func useKeyring() bool {
	if os.Getenv(fileEnvFlag) == "1" {
		return false
	}
	cfg, err := readConfigFile()
	return err != nil || cfg.Store != storeFile
}

// UseFileStore(true) makes the CLI keep its secrets in the config file
// (readable by this user only) from now on, and never touch the keyring;
// UseFileStore(false) goes back to the keyring. Secrets already stored are
// not moved, but a secret missing from the keyring is still looked for in
// the file.
func UseFileStore(on bool) error {
	if _, err := ensureDir(); err != nil {
		return err
	}
	cfg, err := readConfigFile()
	if err != nil {
		return err
	}
	cfg.Store = ""
	if on {
		cfg.Store = storeFile
	}
	return writeConfigFile(cfg)
}

func SaveGitHubToken(token string) error {
	return saveSecret(githubKeyringUser, token, func(c *Config) *string { return &c.GitHubToken })
}

func LoadGitHubToken() (string, error) {
	return loadSecret(githubKeyringUser, ErrNoGitHubToken, func(c *Config) string { return c.GitHubToken })
}

func DeleteGitHubToken() error {
	return deleteSecret(githubKeyringUser, func(c *Config) *string { return &c.GitHubToken })
}

// SaveMaestroKey keeps the key server minted, and which server that was.
func SaveMaestroKey(server, key string) error {
	if err := saveSecret(maestroKeyringUser, key, func(c *Config) *string { return &c.MaestroKey }); err != nil {
		return err
	}
	return saveFieldToFile(func(c *Config) *string { return &c.Server }, server)
}

// LoadMaestroKey returns the key, if it is server's: a key is never sent to
// a server other than the one that minted it. server "" takes it whatever
// server it is for, and a key stored before servers were goes to any.
func LoadMaestroKey(server string) (string, error) {
	key, err := loadSecret(maestroKeyringUser, ErrNoMaestroKey, func(c *Config) string { return c.MaestroKey })
	if err != nil {
		return "", err
	}
	stored, err := Server()
	if err != nil {
		return "", err
	}
	if server != "" && stored != "" && !SameServer(stored, server) {
		return "", &OtherServerError{Stored: stored, Asked: server}
	}
	return key, nil
}

// SaveServer keeps the server to talk to, before any key is minted by it.
func SaveServer(server string) error {
	return saveFieldToFile(func(c *Config) *string { return &c.Server }, server)
}

// Server is the server kept: the one the stored key was minted by, or
// the one asked for before signing in; "" when there is none.
func Server() (string, error) {
	cfg, err := readConfigFile()
	return cfg.Server, err
}

// SameServer is whether two URLs name one server, as typed or as stored.
func SameServer(a, b string) bool {
	return strings.EqualFold(strings.TrimRight(a, "/"), strings.TrimRight(b, "/"))
}

// OtherServerError is a key kept for one server when another was asked for.
type OtherServerError struct{ Stored, Asked string }

func (e *OtherServerError) Error() string {
	return fmt.Sprintf("not signed in to %s: the key here is for %s", e.Asked, e.Stored)
}

func DeleteMaestroKey() error {
	return deleteSecret(maestroKeyringUser, func(c *Config) *string { return &c.MaestroKey })
}

func Reset() error {
	cfg, err := readConfigFile()
	if err != nil {
		return err
	}

	if useKeyring() {
		_ = keyring.Delete(service, maestroKeyringUser)
		_ = keyring.Delete(service, githubKeyringUser)
		for _, f := range cfg.Forges {
			_ = keyring.Delete(service, forgeKeyringPrefix+f.Name)
		}
	}

	p, err := path()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing config file %s: %w", p, err)
	}
	return nil
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

func saveSecret(keyringUser, value string, field func(*Config) *string) error {
	if useKeyring() {
		if err := keyring.Set(service, keyringUser, value); err == nil {
			return clearFileField(field)
		}
	}
	return saveFieldToFile(field, value)
}

func loadSecret(keyringUser string, notFoundErr error, field func(*Config) string) (string, error) {
	if useKeyring() {
		value, err := keyring.Get(service, keyringUser)
		if err == nil {
			return value, nil
		}
	}
	return loadFieldFromFile(notFoundErr, field)
}

func deleteSecret(keyringUser string, field func(*Config) *string) error {
	_ = keyring.Delete(service, keyringUser)
	return clearFileField(field)
}

// read-modify-write so saving one secret never clobbers the other.
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

func writeConfigFile(cfg Config) error {
	p, err := path()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	// os.WriteFile only applies the mode on creation, hence the explicit chmod.
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return fmt.Errorf("writing config file %s: %w", p, err)
	}
	if err := os.Chmod(p, 0o600); err != nil {
		return fmt.Errorf("setting permissions on %s: %w", p, err)
	}
	return nil
}
