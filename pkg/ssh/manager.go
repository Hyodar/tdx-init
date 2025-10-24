package ssh

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"tdx-init/pkg/config"
	"tdx-init/pkg/disks"
)

type Manager struct {
	config      config.SSHConfig
	diskManager *disks.Manager
	provider    KeyProvider
}

type KeyProvider interface {
	WaitForKey(ctx context.Context) (string, error)
}

func NewManager(cfg config.SSHConfig, dm *disks.Manager) (*Manager, error) {
	provider, err := CreateKeyProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH key provider: %w", err)
	}

	return &Manager{
		config:      cfg,
		diskManager: dm,
		provider:    provider,
	}, nil
}

func (sm *Manager) Setup(ctx context.Context) error {
	var sshKey string
	var err error

	if sm.config.StoreAt != "" {
		sshKey, err = sm.tryGetStoredKey()
		if err != nil {
			log.Printf("No stored SSH key found: %v", err)
		}
	}

	if sshKey == "" {
		sshKey, err = sm.waitForKey(ctx)
		if err != nil {
			return fmt.Errorf("failed to get SSH key: %w", err)
		}

		if sm.config.StoreAt != "" {
			if err := sm.storeKeyInDisk(sshKey); err != nil {
				log.Printf("Warning: Failed to store SSH key in disk: %v", err)
			}
		}
	}

	if err := sm.writeSSHKey(sshKey); err != nil {
		return fmt.Errorf("failed to write SSH key: %w", err)
	}

	log.Println("SSH setup completed successfully")
	return nil
}

func (sm *Manager) tryGetStoredKey() (string, error) {
	disk, ok := sm.diskManager.GetDisk(sm.config.StoreAt)
	if !ok {
		return "", fmt.Errorf("disk %s not found", sm.config.StoreAt)
	}

	if disk.DevicePath == "" {
		return "", fmt.Errorf("disk %s not yet initialized", sm.config.StoreAt)
	}

	key, err := disks.GetSSHToken(disk.DevicePath)
	if err != nil {
		return "", err
	}

	log.Printf("Retrieved SSH key from disk %s", sm.config.StoreAt)
	return key, nil
}

func (sm *Manager) storeKeyInDisk(sshKey string) error {
	disk, ok := sm.diskManager.GetDisk(sm.config.StoreAt)
	if !ok {
		return fmt.Errorf("disk %s not found", sm.config.StoreAt)
	}

	if disk.DevicePath == "" {
		return fmt.Errorf("disk %s not yet initialized", sm.config.StoreAt)
	}

	if err := disks.StoreSSHToken(disk.DevicePath, sshKey); err != nil {
		return fmt.Errorf("failed to store SSH token: %w", err)
	}

	log.Printf("Stored SSH key in disk %s", sm.config.StoreAt)
	return nil
}

func (sm *Manager) waitForKey(ctx context.Context) (string, error) {
	return sm.provider.WaitForKey(ctx)
}

func (sm *Manager) writeSSHKey(sshKey string) error {
	if err := os.MkdirAll(sm.config.Dir, 0700); err != nil {
		return fmt.Errorf("failed to create SSH directory: %w", err)
	}

	authKeysFile := filepath.Join(sm.config.Dir, "authorized_keys")
	content := fmt.Sprintf("no-port-forwarding,no-agent-forwarding,no-X11-forwarding ssh-ed25519 %s\n", sshKey)

	if err := os.WriteFile(authKeysFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write authorized_keys: %w", err)
	}

	if sm.config.KeyPath != "" {
		if err := os.WriteFile(sm.config.KeyPath, []byte(sshKey), 0600); err != nil {
			return fmt.Errorf("failed to write key file: %w", err)
		}
	}

	log.Printf("SSH key written to %s", authKeysFile)
	return nil
}

func CreateKeyProvider(cfg config.SSHConfig) (KeyProvider, error) {
	switch cfg.Strategy {
	case "webserver":
		serverURL := ":8080"
		if url, ok := cfg.StrategyConfig["server_url"].(string); ok {
			serverURL = url
		}
		return NewWebServerProvider(serverURL), nil

	default:
		return nil, fmt.Errorf("unknown SSH strategy: %s", cfg.Strategy)
	}
}
