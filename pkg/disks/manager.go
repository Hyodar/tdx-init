package disks

import (
	"context"
	"fmt"
	"log"
	"tdx-init/pkg/config"
	"tdx-init/pkg/keys"
)

type Manager struct {
	disks      map[string]*ManagedDisk
	keyManager *keys.Manager
}

type ManagedDisk struct {
	Name         string
	Config       config.DiskConfig
	DevicePath   string
	MapperName   string
	MapperDevice string
	Initialized  bool
}

func NewManager(cfg *config.Config, km *keys.Manager) (*Manager, error) {
	dm := &Manager{
		disks:      make(map[string]*ManagedDisk),
		keyManager: km,
	}

	for name, diskCfg := range cfg.Disks {
		mapperName := fmt.Sprintf("crypt_%s", name)
		md := &ManagedDisk{
			Name:         name,
			Config:       diskCfg,
			MapperName:   mapperName,
			MapperDevice: fmt.Sprintf("/dev/mapper/%s", mapperName),
			Initialized:  false,
		}
		dm.disks[name] = md
	}

	return dm, nil
}

func (dm *Manager) SetupDisk(ctx context.Context, name string) error {
	disk, ok := dm.disks[name]
	if !ok {
		return fmt.Errorf("disk %s not found", name)
	}

	// Find the physical device
	devicePath, err := dm.findDevice(disk.Config)
	if err != nil {
		return fmt.Errorf("failed to find device for disk %s: %w", name, err)
	}
	disk.DevicePath = devicePath

	log.Printf("Setting up disk %s at device %s", name, devicePath)

	// Check if device has LUKS
	isLuks := IsLuksDevice(devicePath)
	if isLuks {
		disk.Initialized = IsInitialized(devicePath)
		log.Printf("Found existing LUKS container on %s (initialized: %v)", devicePath, disk.Initialized)
	}

	// Determine if we should format
	shouldFormat := dm.shouldFormat(disk, isLuks)
	
	if shouldFormat {
		if err := dm.formatDisk(ctx, disk); err != nil {
			return fmt.Errorf("failed to format disk %s: %w", name, err)
		}
	} else if isLuks {
		if err := dm.mountExistingDisk(ctx, disk); err != nil {
			return fmt.Errorf("failed to mount existing disk %s: %w", name, err)
		}
	} else if disk.Config.EncryptionKey == "" {
		if err := dm.mountPlainDisk(disk); err != nil {
			return fmt.Errorf("failed to mount plain disk %s: %w", name, err)
		}
	} else {
		return fmt.Errorf("disk %s requires formatting but format strategy prevents it", name)
	}

	return nil
}

func (dm *Manager) GetDisk(name string) (*ManagedDisk, bool) {
	disk, ok := dm.disks[name]
	return disk, ok
}

func (dm *Manager) findDevice(cfg config.DiskConfig) (string, error) {
	finder, err := CreateDiskFinder(cfg)
	if err != nil {
		return "", err
	}
	return finder.Find()
}

func (dm *Manager) shouldFormat(disk *ManagedDisk, isLuks bool) bool {
	switch disk.Config.Format {
	case "always":
		return true
	case "never":
		return false
	case "on_initialize":
		if !isLuks {
			// No LUKS means uninitialized, format if encryption is requested
			return disk.Config.EncryptionKey != ""
		}
		// Has LUKS, only format if not initialized
		return !disk.Initialized
	default:
		return false
	}
}

func (dm *Manager) formatDisk(ctx context.Context, disk *ManagedDisk) error {
	if disk.Config.EncryptionKey == "" {
		return dm.formatPlainDisk(disk)
	}

	// Get encryption passphrase
	passphrase, err := dm.keyManager.GetKey(ctx, disk.Config.EncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to get encryption key: %w", err)
	}

	// Format with LUKS
	if err := FormatLuks(disk.DevicePath, passphrase); err != nil {
		return err
	}

	// Store initialization token
	if err := StoreInitToken(disk.DevicePath, disk.Name); err != nil {
		log.Printf("Warning: Failed to store init token: %v", err)
	}

	// Open LUKS device
	if err := OpenLuks(disk.DevicePath, disk.MapperName, passphrase); err != nil {
		return err
	}

	// Create filesystem
	if err := CreateFilesystem(disk.MapperDevice); err != nil {
		CloseLuks(disk.MapperName)
		return err
	}

	// Mount the device
	if err := MountDevice(disk.MapperDevice, disk.Config.MountAt); err != nil {
		CloseLuks(disk.MapperName)
		return fmt.Errorf("failed to mount: %w", err)
	}

	// Create subdirectories
	if err := CreateMountDirs(disk.Config.MountAt, []string{"ssh", "data", "logs"}); err != nil {
		log.Printf("Warning: Failed to create subdirectories: %v", err)
	}

	disk.Initialized = true
	log.Printf("Successfully formatted and mounted encrypted disk %s", disk.Name)
	return nil
}

func (dm *Manager) formatPlainDisk(disk *ManagedDisk) error {
	log.Printf("Formatting plain disk %s", disk.DevicePath)
	
	// Create filesystem
	if err := CreateFilesystem(disk.DevicePath); err != nil {
		return err
	}

	// Mount the device
	if err := MountDevice(disk.DevicePath, disk.Config.MountAt); err != nil {
		return err
	}

	// Create subdirectories
	if err := CreateMountDirs(disk.Config.MountAt, []string{"data", "logs"}); err != nil {
		log.Printf("Warning: Failed to create subdirectories: %v", err)
	}

	disk.Initialized = true
	log.Printf("Successfully formatted and mounted plain disk %s", disk.Name)
	return nil
}

func (dm *Manager) mountExistingDisk(ctx context.Context, disk *ManagedDisk) error {
	if disk.Config.EncryptionKey == "" {
		return fmt.Errorf("encrypted disk %s requires encryption key", disk.Name)
	}

	// Get passphrase
	passphrase, err := dm.keyManager.GetKey(ctx, disk.Config.EncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to get encryption key: %w", err)
	}

	log.Printf("Opening existing LUKS device %s", disk.DevicePath)
	
	// Open LUKS device
	if err := OpenLuks(disk.DevicePath, disk.MapperName, passphrase); err != nil {
		return err
	}

	// Mount the device
	if err := MountDevice(disk.MapperDevice, disk.Config.MountAt); err != nil {
		CloseLuks(disk.MapperName)
		return fmt.Errorf("failed to mount: %w", err)
	}

	log.Printf("Successfully mounted existing encrypted disk %s", disk.Name)
	return nil
}

func (dm *Manager) mountPlainDisk(disk *ManagedDisk) error {
	log.Printf("Mounting plain disk %s", disk.DevicePath)
	
	// Mount the device
	if err := MountDevice(disk.DevicePath, disk.Config.MountAt); err != nil {
		return err
	}

	log.Printf("Successfully mounted plain disk %s", disk.Name)
	return nil
}