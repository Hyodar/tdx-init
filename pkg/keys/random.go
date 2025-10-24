package keys

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"tdx-init/pkg/tpm"
)

type RandomProvider struct {
	Size       int
	UseTPM     bool
	tpmStorage *tpm.TPMStorage
	cachedKey  string
}

func NewRandomProvider(size int, useTPM bool) *RandomProvider {
	return &RandomProvider{
		Size:       size,
		UseTPM:     useTPM,
		tpmStorage: tpm.NewTPMStorage(),
	}
}

func (r *RandomProvider) Get(ctx context.Context) (string, error) {
	if r.UseTPM && r.tpmStorage.Available() {
		key, err := r.tpmStorage.Retrieve()
		if err == nil && key != "" {
			log.Println("Retrieved existing key from TPM")
			r.cachedKey = key
			return key, nil
		}
		log.Printf("No existing key in TPM, generating new one: %v", err)
	}

	if r.cachedKey != "" {
		return r.cachedKey, nil
	}

	key, err := r.generateKey()
	if err != nil {
		return "", err
	}

	r.cachedKey = key

	if r.UseTPM && r.tpmStorage.Available() {
		if err := r.tpmStorage.Store(key); err != nil {
			log.Printf("Warning: Failed to store key in TPM: %v", err)
		}
	}

	return key, nil
}

func (r *RandomProvider) Store(key string) error {
	r.cachedKey = key
	if r.UseTPM && r.tpmStorage.Available() {
		return r.tpmStorage.Store(key)
	}
	return nil
}

func (r *RandomProvider) generateKey() (string, error) {
	var key []byte
	
	if _, err := os.Stat("/dev/hwrng"); err == nil {
		log.Println("Using hardware RNG for key generation")
		file, err := os.Open("/dev/hwrng")
		if err == nil {
			defer file.Close()
			key = make([]byte, r.Size)
			if _, err := file.Read(key); err != nil {
				log.Printf("Failed to read from hardware RNG: %v", err)
				key = nil
			}
		}
	}

	if key == nil {
		log.Println("Using crypto/rand for key generation")
		key = make([]byte, r.Size)
		if _, err := rand.Read(key); err != nil {
			return "", fmt.Errorf("failed to generate random key: %w", err)
		}
	}

	return base64.StdEncoding.EncodeToString(key), nil
}