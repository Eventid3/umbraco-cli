package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	AppName    = "umbraco-cli"
	ConfigFile = "config.toml"
)

// Profile holds credentials for a single Umbraco instance.
type Profile struct {
	BaseURL      string `mapstructure:"base_url"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	Insecure     bool   `mapstructure:"insecure"`
}

// ConfigDir returns ~/.config/umbraco-cli
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", AppName), nil
}

// Load reads the config file and returns a configured Viper instance.
func Load() (*viper.Viper, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(dir)

	// Allow env var overrides: UMBRACO_PROFILE, UMBRACO_BASE_URL, etc.
	v.SetEnvPrefix("UMBRACO")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
		// No config file yet — that's fine on first run.
	}

	return v, nil
}

// GetProfile returns the named profile from the config.
func GetProfile(v *viper.Viper, name string) (*Profile, error) {
	key := fmt.Sprintf("profiles.%s", name)
	if !v.IsSet(key) {
		return nil, fmt.Errorf("profile %q not found in config", name)
	}
	var p Profile
	if err := v.UnmarshalKey(key, &p); err != nil {
		return nil, fmt.Errorf("cannot parse profile %q: %w", name, err)
	}
	if p.BaseURL == "" || p.ClientID == "" || p.ClientSecret == "" {
		return nil, fmt.Errorf("profile %q is incomplete (base_url, client_id, client_secret required)", name)
	}
	return &p, nil
}

// SaveProfile writes a profile to the config file on disk.
func SaveProfile(name string, p Profile) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	// Load existing config (or start fresh).
	v := viper.New()
	v.SetConfigType("toml")
	cfgPath := filepath.Join(dir, ConfigFile)
	v.SetConfigFile(cfgPath)
	_ = v.ReadInConfig() // ignore not-found

	key := fmt.Sprintf("profiles.%s", name)
	v.Set(key+".base_url", p.BaseURL)
	v.Set(key+".client_id", p.ClientID)
	v.Set(key+".client_secret", p.ClientSecret)
	v.Set(key+".insecure", p.Insecure)

	// Set as default if it's the first profile.
	if v.GetString("default_profile") == "" {
		v.Set("default_profile", name)
	}

	return v.WriteConfigAs(cfgPath)
}

// RemoveProfile removes a profile from the config file.
func RemoveProfile(name string) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(dir, ConfigFile)

	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigFile(cfgPath)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("cannot read config: %w", err)
	}

	all := v.AllSettings()
	profiles, ok := all["profiles"].(map[string]interface{})
	if !ok || profiles[name] == nil {
		return fmt.Errorf("profile %q not found", name)
	}
	delete(profiles, name)
	all["profiles"] = profiles

	// If removed profile was the default, clear it.
	if v.GetString("default_profile") == name {
		all["default_profile"] = ""
	}

	newV := viper.New()
	newV.SetConfigType("toml")
	newV.SetConfigFile(cfgPath)
	for k, val := range all {
		newV.Set(k, val)
	}
	return newV.WriteConfigAs(cfgPath)
}

// ListProfiles returns all profile names from the config.
func ListProfiles(v *viper.Viper) []string {
	profiles := v.GetStringMap("profiles")
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	return names
}
