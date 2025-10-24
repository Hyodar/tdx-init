package disks

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"tdx-init/pkg/config"
)

type DiskFinder interface {
	Find() (string, error)
}

func CreateDiskFinder(cfg config.DiskConfig) (DiskFinder, error) {
	switch cfg.Strategy {
	case "largest":
		return NewLargestDiskFinder(), nil

	case "pathglob":
		pattern := "/dev/sd*"
		if p, ok := cfg.StrategyConfig["path_glob"].(string); ok {
			pattern = p
		}
		return NewPathGlobFinder(pattern), nil

	default:
		return nil, fmt.Errorf("unknown disk strategy: %s", cfg.Strategy)
	}
}

func FindFirstDiskByPathGlob(path string) (string, error) {
	disks, err := filepath.Glob(fmt.Sprintf("/dev/disk/by-path/%s", path))
	if err != nil {
		return "", fmt.Errorf("failed to find disk by path: %w", err)
	}

	if len(disks) == 0 {
		return "", fmt.Errorf("no disk found by path: %s", path)
	}

	return disks[0], nil
}

func FindDiskByGlob(pattern string) (string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid glob pattern: %w", err)
	}

	for _, path := range matches {
		if !strings.HasPrefix(path, "/dev/") {
			continue
		}

		if isBootDevice(path) {
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if info.Mode()&os.ModeCharDevice == 0 {
			return path, nil
		}
	}

	return "", fmt.Errorf("no disk found matching pattern %s", pattern)
}

func isBootDevice(device string) bool {
	cmd := exec.Command("findmnt", "-n", "-o", "SOURCE", "/")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	bootDevice := strings.TrimSpace(string(output))
	return strings.HasPrefix(bootDevice, device)
}

func FindLargestDisk() (string, error) {
	file, err := os.Open("/proc/partitions")
	if err != nil {
		return "", err
	}
	defer file.Close()

	var largestDevice string
	var largestSize int64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "major") || line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		deviceName := fields[3]

		hasSCSI := strings.HasPrefix(deviceName, "sd")
		hasSdNumber := strings.ContainsAny(deviceName[len(deviceName)-1:], "0123456789")

		if !hasSCSI || hasSdNumber {
			continue
		}

		sizeBlocks, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			continue
		}

		sizeBytes := sizeBlocks * 1024

		if sizeBytes > largestSize {
			largestSize = sizeBytes
			largestDevice = "/dev/" + deviceName
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if largestDevice == "" {
		return "", fmt.Errorf("no SCSI disk found")
	}

	return largestDevice, nil
}
