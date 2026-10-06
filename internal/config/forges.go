package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"

	"github.com/zalando/go-keyring"
)

var forgeNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

var ErrNoForgeToken = errors.New("no token stored for this forge")

const legacyForgeName = "github"

// Forges returns the configured forges. The first call after upgrading from
// a CLI that only knew GitHub turns its stored token into a forge named
// "github", so nobody has to log in again.
func Forges() ([]Forge, error) {
	cfg, err := readConfigFile()
	if err != nil {
		return nil, err
	}
	if len(cfg.Forges) > 0 {
		return cfg.Forges, nil
	}

	token, err := LoadGitHubToken()
	if err != nil {
		return nil, nil
	}
	if err := AddForge(Forge{Name: legacyForgeName, Kind: "github"}, token); err != nil {
		return nil, fmt.Errorf("migrating the stored GitHub token: %w", err)
	}
	if err := DeleteGitHubToken(); err != nil {
		return nil, fmt.Errorf("migrating the stored GitHub token: %w", err)
	}
	return []Forge{{Name: legacyForgeName, Kind: "github"}}, nil
}

func AddForge(f Forge, token string) error {
	if !forgeNamePattern.MatchString(f.Name) {
		return fmt.Errorf("invalid forge name %q: use lowercase letters, digits, '.', '_' or '-'", f.Name)
	}
	if err := saveForgeToken(f.Name, token); err != nil {
		return err
	}

	if _, err := ensureDir(); err != nil {
		return err
	}
	cfg, err := readConfigFile()
	if err != nil {
		return err
	}
	if i := slices.IndexFunc(cfg.Forges, func(existing Forge) bool { return existing.Name == f.Name }); i >= 0 {
		cfg.Forges[i] = f
	} else {
		cfg.Forges = append(cfg.Forges, f)
	}
	return writeConfigFile(cfg)
}

func RemoveForge(name string) (bool, error) {
	cfg, err := readConfigFile()
	if err != nil {
		return false, err
	}
	i := slices.IndexFunc(cfg.Forges, func(existing Forge) bool { return existing.Name == name })
	if i < 0 {
		return false, nil
	}

	_ = keyring.Delete(service, forgeKeyringPrefix+name)
	cfg.Forges = slices.Delete(cfg.Forges, i, i+1)
	delete(cfg.ForgeTokens, name)
	return true, writeConfigFile(cfg)
}

func ForgeToken(name string) (string, error) {
	if os.Getenv(fileEnvFlag) != "1" {
		if token, err := keyring.Get(service, forgeKeyringPrefix+name); err == nil {
			return token, nil
		}
	}
	cfg, err := readConfigFile()
	if err != nil {
		return "", err
	}
	if token := cfg.ForgeTokens[name]; token != "" {
		return token, nil
	}
	return "", fmt.Errorf("forge %q: %w", name, ErrNoForgeToken)
}

// saveForgeToken prefers the keyring and falls back to the config file,
// like the Maestro key does, keeping only one copy of the token.
func saveForgeToken(name, token string) error {
	inKeyring := false
	if os.Getenv(fileEnvFlag) != "1" {
		inKeyring = keyring.Set(service, forgeKeyringPrefix+name, token) == nil
	}

	if _, err := ensureDir(); err != nil {
		return err
	}
	cfg, err := readConfigFile()
	if err != nil {
		return err
	}
	if inKeyring {
		if _, stored := cfg.ForgeTokens[name]; !stored {
			return nil
		}
		delete(cfg.ForgeTokens, name)
	} else {
		if cfg.ForgeTokens == nil {
			cfg.ForgeTokens = map[string]string{}
		}
		cfg.ForgeTokens[name] = token
	}
	return writeConfigFile(cfg)
}
