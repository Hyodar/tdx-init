package disks

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CreateFilesystem(device string) error {
	log.Printf("Creating ext4 filesystem on %s", device)
	if err := exec.Command("mkfs.ext4", device).Run(); err != nil {
		return fmt.Errorf("failed to create filesystem: %w", err)
	}
	return nil
}

func MountDevice(device, mountPoint string) error {
	if IsMounted(mountPoint) {
		log.Printf("Device already mounted at %s", mountPoint)
		return nil
	}

	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	if err := exec.Command("mount", device, mountPoint).Run(); err != nil {
		return fmt.Errorf("failed to mount device: %w", err)
	}

	return nil
}

func UnmountDevice(mountPoint string) error {
	if !IsMounted(mountPoint) {
		return nil
	}
	return exec.Command("umount", mountPoint).Run()
}

func IsMounted(mountPoint string) bool {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), " "+mountPoint+" ")
}

func CreateMountDirs(mountPoint string, dirs []string) error {
	for _, dir := range dirs {
		fullPath := filepath.Join(mountPoint, dir)
		if err := os.MkdirAll(fullPath, 0700); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
		}
	}
	return nil
}