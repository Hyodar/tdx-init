package tdxsetup

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"syscall"
)

type PassphraseInitializerer interface {
	WaitForPassphrase(ctx context.Context) ([]byte, error)
}

type RandomPassphraseInitializer struct{}

func (r *RandomPassphraseInitializer) WaitForPassphrase(_ context.Context) ([]byte, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}
	return randomBytes, nil
}

type NamedPipePassphraseInitializer struct {
	PipePath string
}

func (n *NamedPipePassphraseInitializer) WaitForPassphrase(_ context.Context) ([]byte, error) {
	if err := syscall.Mkfifo(n.PipePath, 0600); err != nil {
		if !os.IsExist(err) {
			return nil, fmt.Errorf("failed to create named pipe: %w", err)
		}
	}

	file, err := os.OpenFile(n.PipePath, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open named pipe: %w", err)
	}
	defer file.Close()

	passphrase := make([]byte, 1024)
	length, err := file.Read(passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to read from named pipe: %w", err)
	}

	result := passphrase[:length]
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}
