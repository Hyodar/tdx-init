package tdxsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"tdx-init/pkg/disk"
)

type TdxSetupManager struct {
	keyInitializer        KeyInitializerer
	passphraseInitializer PassphraseInitializerer
	diskInitializer       DiskInitializerer
	options               *TdxSetupManagerOptions
}

type TdxSetupManagerOptions struct {
	SSHDir       string
	KeyFile      string
	MountPoint   string
	MapperName   string
	MapperDevice string
}

func DefaultTdxSetupManagerOptions() *TdxSetupManagerOptions {
	return &TdxSetupManagerOptions{
		SSHDir:       "/root/.ssh",
		KeyFile:      "/etc/root_key",
		MountPoint:   "/persistent",
		MapperName:   "cryptdisk",
		MapperDevice: "/dev/mapper/cryptdisk",
	}
}

func NewSetupManager(
	keyInitializer KeyInitializerer,
	passphraseInitializer PassphraseInitializerer,
	diskInitializer DiskInitializerer,
	options *TdxSetupManagerOptions,
) *TdxSetupManager {
	return &TdxSetupManager{
		keyInitializer:        keyInitializer,
		passphraseInitializer: passphraseInitializer,
		diskInitializer:       diskInitializer,
		options:               options,
	}
}

func (m *TdxSetupManager) Setup(ctx context.Context) error {
	devicePath, err := m.diskInitializer.FindDisk(ctx)
	if err != nil {
		return fmt.Errorf("failed to find disk: %w", err)
	}

	key, err := m.tryFindKey(ctx, devicePath)
	if err != nil {
		return fmt.Errorf("failed to find key: %w", err)
	}

	if err := m.writeKey(key); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}

	passphrase, err := m.passphraseInitializer.WaitForPassphrase(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for passphrase: %w", err)
	}

	if err := disk.SetPassphrase(devicePath, m.options.KeyFile, m.options.MapperName, m.options.MapperDevice, m.options.MountPoint, string(passphrase)); err != nil {
		return fmt.Errorf("failed to set passphrase: %w", err)
	}

	return nil
}

func (m *TdxSetupManager) tryFindKey(ctx context.Context, devicePath string) (string, error) {
	cmd := exec.Command("cryptsetup", "isLuks", devicePath)
	if cmd.Run() == nil {
		log.Println("Found existing LUKS container, extracting key...")
		cmd := exec.Command("cryptsetup", "token", "export", "--token-id", "1", devicePath)
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to export LUKS token: %w", err)
		}

		var token disk.Token
		if err := json.Unmarshal(output, &token); err != nil {
			return "", fmt.Errorf("failed to parse token JSON: %w", err)
		}

		keyData, ok := token.UserData["metadata"]
		if !ok {
			return "", fmt.Errorf("no metadata found in token")
		}

		return string(keyData), nil
	}

	log.Printf("No LUKS container found, waiting for key...")

	key, err := m.keyInitializer.WaitForKey(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to wait for key: %w", err)
	}

	return key, nil
}

func (m *TdxSetupManager) writeKey(key string) error {
	os.MkdirAll(m.options.SSHDir, 0700)

	if err := os.Chown(m.options.SSHDir, 1000, 1000); err != nil {
		log.Printf("Warning: Could not set ownership on .ssh dir: %v", err)
	}

	authKeysFile := filepath.Join(m.options.SSHDir, "authorized_keys")
	f, err := os.OpenFile(authKeysFile, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("error opening authorized_keys: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString("no-port-forwarding,no-agent-forwarding,no-X11-forwarding ssh-ed25519 " + key + "\n"); err != nil {
		return fmt.Errorf("error writing to authorized_keys: %w", err)
	}

	// if err := os.Chown(authKeysFile, 1000, 1000); err != nil {
	// 	log.Printf("Warning: Could not set ownership on authorized_keys: %v", err)
	// }

	if err := os.WriteFile(m.options.KeyFile, []byte(key), 0600); err != nil {
		return fmt.Errorf("error writing key file: %w", err)
	}

	return nil
}
