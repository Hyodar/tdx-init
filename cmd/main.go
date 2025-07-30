package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"tdx-init/pkg/tdxsetup"

	"github.com/spf13/cobra"
)

type Config struct {
	KeyStrategy struct {
		Type      string
		ServerURL string
	}
	PassphraseStrategy struct {
		Type     string
		PipePath string
	}
	DiskStrategy struct {
		Type     string
		PathGlob string
	}
	SSHDir       string
	KeyFile      string
	MountPoint   string
	MapperName   string
	MapperDevice string
}

var config Config

var rootCmd = &cobra.Command{
	Use:   "tdx-init",
	Short: "TDX Init CLI - Trusted Device Setup",
	Long: `A CLI tool for setting up trusted device initialization with configurable
strategies for key management, passphrase handling, and disk encryption.`,
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run the TDX setup process",
	Long: `Runs the complete TDX setup process using the configured strategies for
key initialization, passphrase generation, and disk management.`,
	Run: func(cmd *cobra.Command, args []string) {
		runSetup(config)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Long:  `Displays the current configuration settings for all strategies and options.`,
	Run: func(cmd *cobra.Command, args []string) {
		showConfig(config)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&config.SSHDir, "ssh-dir", "/root/.ssh", "SSH directory path")
	rootCmd.PersistentFlags().StringVar(&config.KeyFile, "key-file", "/etc/root_key", "Key file path")
	rootCmd.PersistentFlags().StringVar(&config.MountPoint, "mount-point", "/persistent", "Mount point for encrypted disk")
	rootCmd.PersistentFlags().StringVar(&config.MapperName, "mapper-name", "cryptdisk", "Device mapper name")
	rootCmd.PersistentFlags().StringVar(&config.MapperDevice, "mapper-device", "/dev/mapper/cryptdisk", "Device mapper path")

	setupCmd.Flags().StringVar(&config.KeyStrategy.Type, "key-strategy.type", "webserver", "Key initialization strategy (webserver)")
	setupCmd.Flags().StringVar(&config.KeyStrategy.ServerURL, "key-strategy.server-url", "0.0.0.0:8080", "URL for webserver key strategy")
	setupCmd.Flags().StringVar(&config.PassphraseStrategy.Type, "passphrase-strategy.type", "random", "Passphrase strategy (random, namedpipe)")
	setupCmd.Flags().StringVar(&config.PassphraseStrategy.PipePath, "passphrase-strategy.pipe-path", "/tmp/passphrase", "Path for named pipe passphrase strategy")
	setupCmd.Flags().StringVar(&config.DiskStrategy.Type, "disk-strategy.type", "largest", "Disk strategy (largest, pathglob)")
	setupCmd.Flags().StringVar(&config.DiskStrategy.PathGlob, "disk-strategy.path-glob", "*", "Path glob for pathglob disk strategy")

	configCmd.Flags().StringVar(&config.KeyStrategy.Type, "key-strategy.type", "webserver", "Key initialization strategy (webserver)")
	configCmd.Flags().StringVar(&config.KeyStrategy.ServerURL, "key-strategy.server-url", ":8080", "URL for webserver key strategy")
	configCmd.Flags().StringVar(&config.PassphraseStrategy.Type, "passphrase-strategy.type", "random", "Passphrase strategy (random, namedpipe)")
	configCmd.Flags().StringVar(&config.PassphraseStrategy.PipePath, "passphrase-strategy.pipe-path", "/tmp/passphrase", "Path for named pipe passphrase strategy")
	configCmd.Flags().StringVar(&config.DiskStrategy.Type, "disk-strategy.type", "largest", "Disk strategy (largest, pathglob)")
	configCmd.Flags().StringVar(&config.DiskStrategy.PathGlob, "disk-strategy.path-glob", "*", "Path glob for pathglob disk strategy")

	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(configCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSetup(config Config) {
	ctx := context.Background()

	keyInit, err := createKeyInitializer(config)
	if err != nil {
		log.Fatal("Failed to create key initializer:", err)
	}

	passphraseInit, err := createPassphraseInitializer(config)
	if err != nil {
		log.Fatal("Failed to create passphrase initializer:", err)
	}

	diskInit, err := createDiskInitializer(config)
	if err != nil {
		log.Fatal("Failed to create disk initializer:", err)
	}

	options := &tdxsetup.TdxSetupManagerOptions{
		SSHDir:       config.SSHDir,
		KeyFile:      config.KeyFile,
		MountPoint:   config.MountPoint,
		MapperName:   config.MapperName,
		MapperDevice: config.MapperDevice,
	}

	manager := tdxsetup.NewSetupManager(keyInit, passphraseInit, diskInit, options)

	fmt.Println("Starting TDX setup process...")
	if err := manager.Setup(ctx); err != nil {
		log.Fatal("Setup failed:", err)
	}

	fmt.Println("TDX setup completed successfully!")
}

func createKeyInitializer(config Config) (tdxsetup.KeyInitializerer, error) {
	switch config.KeyStrategy.Type {
	case "webserver":
		return &tdxsetup.WebServerKeyInitializer{URL: config.KeyStrategy.ServerURL}, nil
	default:
		return nil, fmt.Errorf("unknown key strategy: %s", config.KeyStrategy.Type)
	}
}

func createPassphraseInitializer(config Config) (tdxsetup.PassphraseInitializerer, error) {
	switch config.PassphraseStrategy.Type {
	case "random":
		return &tdxsetup.RandomPassphraseInitializer{}, nil
	case "namedpipe":
		return &tdxsetup.NamedPipePassphraseInitializer{PipePath: config.PassphraseStrategy.PipePath}, nil
	default:
		return nil, fmt.Errorf("unknown passphrase strategy: %s", config.PassphraseStrategy.Type)
	}
}

func createDiskInitializer(config Config) (tdxsetup.DiskInitializerer, error) {
	switch config.DiskStrategy.Type {
	case "largest":
		return &tdxsetup.LargestDiskInitializer{}, nil
	case "pathglob":
		return &tdxsetup.PathGlobDiskInitializer{PathGlob: config.DiskStrategy.PathGlob}, nil
	default:
		return nil, fmt.Errorf("unknown disk strategy: %s", config.DiskStrategy.Type)
	}
}

func showConfig(config Config) {
	fmt.Println("Current Configuration:")
	fmt.Printf("  Key Strategy: %s\n", config.KeyStrategy.Type)
	if config.KeyStrategy.Type == "webserver" {
		fmt.Printf("    Server URL: %s\n", config.KeyStrategy.ServerURL)
	}
	fmt.Printf("  Passphrase Strategy: %s\n", config.PassphraseStrategy.Type)
	if config.PassphraseStrategy.Type == "namedpipe" {
		fmt.Printf("    Pipe Path: %s\n", config.PassphraseStrategy.PipePath)
	}
	fmt.Printf("  Disk Strategy: %s\n", config.DiskStrategy.Type)
	if config.DiskStrategy.Type == "pathglob" {
		fmt.Printf("    Path Glob: %s\n", config.DiskStrategy.PathGlob)
	}
	fmt.Printf("  SSH Directory: %s\n", config.SSHDir)
	fmt.Printf("  Key File: %s\n", config.KeyFile)
	fmt.Printf("  Mount Point: %s\n", config.MountPoint)
	fmt.Printf("  Mapper Name: %s\n", config.MapperName)
	fmt.Printf("  Mapper Device: %s\n", config.MapperDevice)
}
