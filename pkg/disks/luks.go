package disks

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

const (
	InitTokenID = "1"
	SSHTokenID  = "2"
)

type Token struct {
	Type     string            `json:"type"`
	Keyslots []string          `json:"keyslots"`
	UserData map[string]string `json:"user_data"`
}

func IsLuksDevice(devicePath string) bool {
	cmd := exec.Command("cryptsetup", "isLuks", devicePath)
	return cmd.Run() == nil
}

func IsInitialized(devicePath string) bool {
	cmd := exec.Command("cryptsetup", "token", "export", "--token-id", InitTokenID, devicePath)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	var token Token
	if err := json.Unmarshal(output, &token); err != nil {
		return false
	}

	return token.UserData["initialized"] == "true"
}

func FormatLuks(devicePath, passphrase string) error {
	log.Printf("Formatting %s with LUKS2 encryption", devicePath)

	cmd := exec.Command("cryptsetup", "luksFormat", "--type", "luks2", "-q", devicePath)
	cmd.Stdin = strings.NewReader(passphrase)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to format with LUKS: %w", err)
	}

	return nil
}

func OpenLuks(devicePath, mapperName, passphrase string) error {
	cmd := exec.Command("cryptsetup", "open", devicePath, mapperName)
	cmd.Stdin = strings.NewReader(passphrase)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open LUKS device: %w", err)
	}
	return nil
}

func CloseLuks(mapperName string) error {
	return exec.Command("cryptsetup", "close", mapperName).Run()
}

func StoreInitToken(devicePath, diskName string) error {
	token := Token{
		Type:     "tdx-init",
		Keyslots: []string{},
		UserData: map[string]string{
			"initialized": "true",
			"disk_name":   diskName,
		},
	}

	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal init token: %w", err)
	}

	cmd := exec.Command("cryptsetup", "token", "import", "--token-id", InitTokenID, devicePath)
	cmd.Stdin = strings.NewReader(string(tokenJSON))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store init token: %w", err)
	}

	return nil
}

func StoreSSHToken(devicePath, sshKey string) error {
	token := Token{
		Type:     "ssh-key",
		Keyslots: []string{},
		UserData: map[string]string{
			"ssh_key": sshKey,
		},
	}

	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal SSH token: %w", err)
	}

	cmd := exec.Command("cryptsetup", "token", "import", "--token-id", SSHTokenID, devicePath)
	cmd.Stdin = strings.NewReader(string(tokenJSON))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store SSH token: %w", err)
	}

	return nil
}

func GetSSHToken(devicePath string) (string, error) {
	cmd := exec.Command("cryptsetup", "token", "export", "--token-id", SSHTokenID, devicePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no SSH token found")
	}

	var token Token
	if err := json.Unmarshal(output, &token); err != nil {
		return "", fmt.Errorf("failed to parse SSH token: %w", err)
	}

	key, ok := token.UserData["ssh_key"]
	if !ok {
		return "", fmt.Errorf("no SSH key in token")
	}

	return key, nil
}
