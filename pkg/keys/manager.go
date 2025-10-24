package keys

import (
	"context"
	"fmt"
	"tdx-init/pkg/config"
)

type Manager struct {
	keys map[string]Provider
}

type Provider interface {
	Get(ctx context.Context) (string, error)
	Store(key string) error
}

func NewManager(cfg *config.Config) (*Manager, error) {
	m := &Manager{
		keys: make(map[string]Provider),
	}

	for name, keyCfg := range cfg.Keys {
		provider, err := CreateProvider(keyCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create key provider for %s: %w", name, err)
		}
		m.keys[name] = provider
	}

	return m, nil
}

func (m *Manager) GetKey(ctx context.Context, name string) (string, error) {
	provider, ok := m.keys[name]
	if !ok {
		return "", fmt.Errorf("key %s not found", name)
	}
	return provider.Get(ctx)
}

func (m *Manager) StoreKey(name string, key string) error {
	provider, ok := m.keys[name]
	if !ok {
		return fmt.Errorf("key %s not found", name)
	}
	return provider.Store(key)
}

func CreateProvider(cfg config.KeyConfig) (Provider, error) {
	switch cfg.Strategy {
	case "random":
		size := 64
		if s, ok := cfg.StrategyConfig["size"].(int); ok {
			size = s
		}
		return NewRandomProvider(size, cfg.TPM), nil

	case "pipe":
		pipePath := "/tmp/passphrase"
		if path, ok := cfg.StrategyConfig["pipe_path"].(string); ok {
			pipePath = path
		}
		return NewPipeProvider(pipePath, cfg.TPM), nil

	default:
		return nil, fmt.Errorf("unknown key strategy: %s", cfg.Strategy)
	}
}
