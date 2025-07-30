package disk

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
