package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SSH   SSHConfig            `yaml:"ssh"`
	Keys  map[string]KeyConfig `yaml:"keys"`
	Disks map[string]DiskConfig `yaml:"disks"`
}

type SSHConfig struct {
	Strategy       string                 `yaml:"strategy"`
	StrategyConfig map[string]interface{} `yaml:"strategy_config"`
	Dir            string                 `yaml:"dir"`
	KeyPath        string                 `yaml:"key_path"`
	StoreAt        string                 `yaml:"store_at"`
}

type KeyConfig struct {
	Strategy       string                 `yaml:"strategy"`
	StrategyConfig map[string]interface{} `yaml:"strategy_config"`
	TPM            bool                   `yaml:"tpm"`
}

type DiskConfig struct {
	Strategy      string                 `yaml:"strategy"`
	StrategyConfig map[string]interface{} `yaml:"strategy_config"`
	Format        string                 `yaml:"format"`
	EncryptionKey string                 `yaml:"encryption_key"`
	MountAt       string                 `yaml:"mount_at"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if c.SSH.Strategy == "" {
		return fmt.Errorf("ssh.strategy is required")
	}
	if c.SSH.Dir == "" {
		c.SSH.Dir = "/root/.ssh"
	}
	if c.SSH.KeyPath == "" {
		c.SSH.KeyPath = "/etc/root_key"
	}

	for name, key := range c.Keys {
		if key.Strategy == "" {
			return fmt.Errorf("keys.%s.strategy is required", name)
		}
		if key.Strategy != "random" && key.Strategy != "pipe" {
			return fmt.Errorf("keys.%s.strategy must be 'random' or 'pipe'", name)
		}
	}

	for name, disk := range c.Disks {
		if disk.Strategy == "" {
			return fmt.Errorf("disks.%s.strategy is required", name)
		}
		if disk.Strategy != "largest" && disk.Strategy != "pathglob" {
			return fmt.Errorf("disks.%s.strategy must be 'largest' or 'pathglob'", name)
		}
		if disk.Format == "" {
			disk.Format = "on_initialize"
		}
		if disk.Format != "always" && disk.Format != "on_initialize" && disk.Format != "never" {
			return fmt.Errorf("disks.%s.format must be 'always', 'on_initialize', or 'never'", name)
		}
		if disk.MountAt == "" {
			return fmt.Errorf("disks.%s.mount_at is required", name)
		}
		c.Disks[name] = disk
	}

	if c.SSH.StoreAt != "" {
		if _, ok := c.Disks[c.SSH.StoreAt]; !ok {
			return fmt.Errorf("ssh.store_at references non-existent disk '%s'", c.SSH.StoreAt)
		}
	}

	for name, disk := range c.Disks {
		if disk.EncryptionKey != "" {
			if _, ok := c.Keys[disk.EncryptionKey]; !ok {
				return fmt.Errorf("disks.%s.encryption_key references non-existent key '%s'", name, disk.EncryptionKey)
			}
		}
	}

	return nil
}