package disk

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Token struct {
	Type     string            `json:"type"`
	Keyslots []string          `json:"keyslots"`
	UserData map[string]string `json:"user_data"`
}

func SetPassphrase(
	devicePath string,
	keyFile string,
	mapperName string,
	mapperDevice string,
	mountPoint string,
	passphrase string,
) error {
	if CheckMounted(mountPoint) {
		return fmt.Errorf("encrypted disk already setup")
	}

	if _, err := os.Stat(keyFile); err != nil {
		return fmt.Errorf("SSH key not set. Provide public key via HTTP first.")
	}

	cmd := exec.Command("cryptsetup", "isLuks", devicePath)
	isNewSetup := cmd.Run() != nil

	if isNewSetup {
		SetupNewDisk(keyFile, devicePath, passphrase, mapperName, mapperDevice, mountPoint)
		// SetupMountDirs(mountPoint, mountDirs)
	} else {
		MountExistingDisk(keyFile, devicePath, passphrase, mapperName, mapperDevice, mountPoint)
	}

	return nil
}

func SetupNewDisk(
	keyFile string,
	devicePath string,
	passphrase string,
	mapperName string,
	mapperDevice string,
	mountPoint string,
) error {
	log.Println("Formatting disk with LUKS2...")
	cmd := exec.Command("cryptsetup", "luksFormat", "--type", "luks2", "-q", devicePath)
	cmd.Stdin = strings.NewReader(passphrase)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error formatting disk: %w", err)
	}

	cmd = exec.Command("cryptsetup", "open", devicePath, mapperName)
	cmd.Stdin = strings.NewReader(passphrase)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error opening LUKS device: %w", err)
	}

	log.Println("Creating ext4 filesystem...")
	if err := exec.Command("mkfs.ext4", mapperDevice).Run(); err != nil {
		exec.Command("cryptsetup", "close", mapperName).Run()
		return fmt.Errorf("error creating filesystem: %w", err)
	}

	os.MkdirAll(mountPoint, 0755)
	if err := exec.Command("mount", mapperDevice, mountPoint).Run(); err != nil {
		exec.Command("cryptsetup", "close", mapperName).Run()
		return fmt.Errorf("error mounting filesystem: %w", err)
	}

	key, err := os.ReadFile(keyFile)
	if err != nil {
		cleanupErr := CleanupMount(mountPoint, mapperName)
		return fmt.Errorf("error reading SSH key file: %w", errors.Join(err, cleanupErr))
	}

	token := Token{
		Type:     "user",
		Keyslots: []string{},
		UserData: map[string]string{
			"metadata": string(key),
		},
	}

	tokenJSON, err := json.Marshal(token)
	if err != nil {
		cleanupErr := CleanupMount(mountPoint, mapperName)
		return fmt.Errorf("error marshaling token JSON: %w", errors.Join(err, cleanupErr))
	}

	log.Println("Saving SSH key...")
	cmd = exec.Command("cryptsetup", "token", "import", "--token-id", "1", devicePath)
	cmd.Stdin = strings.NewReader(string(tokenJSON))

	if err := cmd.Run(); err != nil {
		cleanupErr := CleanupMount(mountPoint, mapperName)
		return fmt.Errorf("error importing token to LUKS header: %w", errors.Join(err, cleanupErr))
	}

	fmt.Println("Encrypted disk initialized and mounted successfully")

	return nil
}

func MountExistingDisk(
	keyFile string,
	devicePath string,
	passphrase string,
	mapperName string,
	mapperDevice string,
	mountPoint string,
) error {
	cmd := exec.Command("cryptsetup", "open", devicePath, mapperName)
	cmd.Stdin = strings.NewReader(passphrase)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error opening LUKS device: %w", err)
	}

	os.MkdirAll(mountPoint, 0755)
	if err := exec.Command("mount", mapperDevice, mountPoint).Run(); err != nil {
		if err := exec.Command("cryptsetup", "close", mapperName).Run(); err != nil {
			return fmt.Errorf("error closing LUKS device: %w", err)
		}
		return fmt.Errorf("error mounting filesystem: %w", err)
	}

	fmt.Println("Encrypted disk mounted successfully")

	return nil
}

func CheckMounted(mountPoint string) bool {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), " "+mountPoint+" ")
}

func CleanupMount(mountPoint string, mapperName string) error {
	if err := exec.Command("umount", mountPoint).Run(); err != nil {
		return fmt.Errorf("error unmounting filesystem: %w", err)
	}
	if err := exec.Command("cryptsetup", "close", mapperName).Run(); err != nil {
		return fmt.Errorf("error closing LUKS device: %w", err)
	}
	return nil
}
