package disks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type PathGlobFinder struct {
	Pattern string
}

func NewPathGlobFinder(pattern string) *PathGlobFinder {
	return &PathGlobFinder{
		Pattern: pattern,
	}
}

func (f *PathGlobFinder) Find() (string, error) {
	matches, err := filepath.Glob(f.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid glob pattern: %w", err)
	}

	for _, path := range matches {
		if !strings.HasPrefix(path, "/dev/") {
			continue
		}

		if f.isBootDevice(path) {
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

	return "", fmt.Errorf("no disk found matching pattern %s", f.Pattern)
}

func (f *PathGlobFinder) isBootDevice(device string) bool {
	cmd := exec.Command("findmnt", "-n", "-o", "SOURCE", "/")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	bootDevice := strings.TrimSpace(string(output))
	return strings.HasPrefix(bootDevice, device)
}