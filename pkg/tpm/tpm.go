package tpm

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

const (
	DefaultNVIndex = "0x1500016"
	TCTIDevice     = "device:/dev/tpmrm0"
)

type TPMStorage struct {
	NVIndex string
}

func NewTPMStorage() *TPMStorage {
	return &TPMStorage{
		NVIndex: DefaultNVIndex,
	}
}

func (t *TPMStorage) Available() bool {
	cmd := exec.Command("test", "-e", "/dev/tpm0")
	if err := cmd.Run(); err != nil {
		return false
	}
	cmd = exec.Command("test", "-e", "/dev/tpmrm0")
	return cmd.Run() == nil
}

func (t *TPMStorage) Store(key string) error {
	if !t.Available() {
		return fmt.Errorf("TPM device not available")
	}

	keySize := len(key)
	
	cmd := exec.Command("tpm2_nvundefine", t.NVIndex)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("TPM2TOOLS_TCTI=%s", TCTIDevice))
	cmd.Run()

	log.Printf("Defining TPM NV index %s with size %d", t.NVIndex, keySize)
	cmd = exec.Command("tpm2_nvdefine", "-s", fmt.Sprintf("%d", keySize), t.NVIndex)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("TPM2TOOLS_TCTI=%s", TCTIDevice))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to define TPM NV index: %w", err)
	}

	log.Printf("Writing key to TPM NV index %s", t.NVIndex)
	cmd = exec.Command("tpm2_nvwrite", t.NVIndex, "-i-")
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("TPM2TOOLS_TCTI=%s", TCTIDevice))
	cmd.Stdin = bytes.NewBufferString(key)
	
	if output, err := cmd.CombinedOutput(); err != nil {
		t.cleanup()
		return fmt.Errorf("failed to write key to TPM: %w (output: %s)", err, string(output))
	}

	log.Printf("Successfully stored key in TPM at index %s", t.NVIndex)
	return nil
}

func (t *TPMStorage) Retrieve() (string, error) {
	if !t.Available() {
		return "", fmt.Errorf("TPM device not available")
	}

	log.Printf("Reading from TPM NV index %s", t.NVIndex)
	cmd := exec.Command("tpm2_nvread", t.NVIndex)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("TPM2TOOLS_TCTI=%s", TCTIDevice))
	
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "ERROR") || strings.Contains(stderr, "not found") {
				return "", fmt.Errorf("no key stored in TPM at index %s", t.NVIndex)
			}
		}
		return "", fmt.Errorf("failed to read from TPM: %w", err)
	}

	key := strings.TrimSpace(string(output))
	if key == "" {
		return "", fmt.Errorf("empty key retrieved from TPM")
	}

	return key, nil
}

func (t *TPMStorage) cleanup() {
	cmd := exec.Command("tpm2_nvundefine", t.NVIndex)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("TPM2TOOLS_TCTI=%s", TCTIDevice))
	cmd.Run()
}

func (t *TPMStorage) Clear() error {
	if !t.Available() {
		return fmt.Errorf("TPM device not available")
	}

	cmd := exec.Command("tpm2_nvundefine", t.NVIndex)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("TPM2TOOLS_TCTI=%s", TCTIDevice))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clear TPM NV index: %w", err)
	}
	
	return nil
}