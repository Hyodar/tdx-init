package keys

import (
	"context"
	"fmt"
	"log"
	"os"
	"syscall"
	"tdx-init/pkg/tpm"
)

type PipeProvider struct {
	PipePath   string
	UseTPM     bool
	tpmStorage *tpm.TPMStorage
	cachedKey  string
}

func NewPipeProvider(pipePath string, useTPM bool) *PipeProvider {
	return &PipeProvider{
		PipePath:   pipePath,
		UseTPM:     useTPM,
		tpmStorage: tpm.NewTPMStorage(),
	}
}

func (p *PipeProvider) Get(ctx context.Context) (string, error) {
	if p.UseTPM && p.tpmStorage.Available() {
		key, err := p.tpmStorage.Retrieve()
		if err == nil && key != "" {
			log.Println("Retrieved existing key from TPM")
			p.cachedKey = key
			return key, nil
		}
	}

	if p.cachedKey != "" {
		return p.cachedKey, nil
	}

	if err := os.MkdirAll("/tmp", 0755); err != nil {
		return "", fmt.Errorf("failed to create /tmp directory: %w", err)
	}

	if _, err := os.Stat(p.PipePath); os.IsNotExist(err) {
		if err := os.Remove(p.PipePath); err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to remove existing pipe: %w", err)
		}
		if err := syscall.Mkfifo(p.PipePath, 0600); err != nil {
			return "", fmt.Errorf("failed to create named pipe: %w", err)
		}
	}

	log.Printf("Waiting for key on named pipe %s", p.PipePath)
	
	keyChan := make(chan string, 1)
	errChan := make(chan error, 1)
	
	go func() {
		data, err := os.ReadFile(p.PipePath)
		if err != nil {
			errChan <- err
			return
		}
		keyChan <- string(data)
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-errChan:
		return "", fmt.Errorf("failed to read from pipe: %w", err)
	case key := <-keyChan:
		p.cachedKey = key
		if p.UseTPM && p.tpmStorage.Available() {
			if err := p.tpmStorage.Store(key); err != nil {
				log.Printf("Warning: Failed to store key in TPM: %v", err)
			}
		}
		return key, nil
	}
}

func (p *PipeProvider) Store(key string) error {
	p.cachedKey = key
	if p.UseTPM && p.tpmStorage.Available() {
		return p.tpmStorage.Store(key)
	}
	return nil
}