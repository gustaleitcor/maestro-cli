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

var ErrNoGitHubToken = errors.New("no github token found; run `maestro login` first")
var ErrNoMaestroKey = errors.New("no maestro key found; run `maestro login` first")

func SaveGitHubToken(token string) error {
	return saveSecret(githubKeyringUser, token, func(c *Config) *string { return &c.GitHubToken })
}

func LoadGitHubToken() (string, error) {
	return loadSecret(githubKeyringUser, ErrNoGitHubToken, func(c *Config) string { return c.GitHubToken })
}

func DeleteGitHubToken() error {
	return deleteSecret(githubKeyringUser, func(c *Config) *string { return &c.GitHubToken })
}

func SaveMaestroKey(key string) error {
	return saveSecret(maestroKeyringUser, key, func(c *Config) *string { return &c.MaestroKey })
}

func LoadMaestroKey() (string, error) {
	return loadSecret(maestroKeyringUser, ErrNoMaestroKey, func(c *Config) string { return c.MaestroKey })
}

func DeleteMaestroKey() error {
	return deleteSecret(maestroKeyringUser, func(c *Config) *string { return &c.MaestroKey })
}

func saveSecret(keyringUser, value string, field func(*Config) *string) error {
	if os.Getenv(fileEnvFlag) != "1" {
		if err := keyring.Set(service, keyringUser, value); err == nil {
			return clearFileField(field)
		}
	}
	return saveFieldToFile(field, value)
}

func loadSecret(keyringUser string, notFoundErr error, field func(*Config) string) (string, error) {
	if os.Getenv(fileEnvFlag) != "1" {
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
