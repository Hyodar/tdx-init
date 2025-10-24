package setup

import (
	"context"
	"fmt"
	"log"
	"tdx-init/pkg/config"
	"tdx-init/pkg/disks"
	"tdx-init/pkg/keys"
	"tdx-init/pkg/ssh"
)

type Orchestrator struct {
	config      *config.Config
	keyManager  *keys.Manager
	diskManager *disks.Manager
	sshManager  *ssh.Manager
}

func NewOrchestrator(cfg *config.Config) (*Orchestrator, error) {
	keyManager, err := keys.NewManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create key manager: %w", err)
	}

	diskManager, err := disks.NewManager(cfg, keyManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create disk manager: %w", err)
	}

	sshManager, err := ssh.NewManager(cfg.SSH, diskManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH manager: %w", err)
	}

	return &Orchestrator{
		config:      cfg,
		keyManager:  keyManager,
		diskManager: diskManager,
		sshManager:  sshManager,
	}, nil
}

func (o *Orchestrator) Setup(ctx context.Context) error {
	log.Println("Starting TDX initialization...")

	disksToSetup := o.getDisksInOrder()
	
	for _, diskName := range disksToSetup {
		log.Printf("Setting up disk: %s", diskName)
		if err := o.diskManager.SetupDisk(ctx, diskName); err != nil {
			return fmt.Errorf("failed to setup disk %s: %w", diskName, err)
		}
	}

	log.Println("Setting up SSH...")
	if err := o.sshManager.Setup(ctx); err != nil {
		return fmt.Errorf("failed to setup SSH: %w", err)
	}

	log.Println("TDX initialization completed successfully")
	return nil
}

func (o *Orchestrator) getDisksInOrder() []string {
	var order []string
	
	if o.config.SSH.StoreAt != "" {
		order = append(order, o.config.SSH.StoreAt)
	}

	for name := range o.config.Disks {
		if name != o.config.SSH.StoreAt {
			order = append(order, name)
		}
	}

	return order
}