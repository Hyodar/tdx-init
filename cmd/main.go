package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"tdx-init/pkg/config"
	"tdx-init/pkg/setup"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "tdx-init",
	Short: "TDX Init - Secure disk encryption and SSH key management",
	Long: `A configurable CLI tool for secure disk encryption and SSH key management
in TDX (Trusted Domain Extensions) environments. Provides flexible strategies
for key initialization, passphrase generation, and disk selection.`,
}

var setupCmd = &cobra.Command{
	Use:   "setup [config]",
	Short: "Run the TDX setup process",
	Long: `Runs the complete TDX setup process using configuration from a YAML file.
This includes disk encryption, SSH key management, and persistent storage setup.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			configFile = args[0]
		}
		runSetup()
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate [config]",
	Short: "Validate configuration file",
	Long:  `Validates the YAML configuration file and displays the parsed configuration.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			configFile = args[0]
		}
		validateConfig()
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(generateConfigCmd)
}

var generateConfigCmd = &cobra.Command{
	Use:   "generate-config",
	Short: "Generate example configuration file",
	Long:  `Generates an example YAML configuration file with all available options.`,
	Run: func(cmd *cobra.Command, args []string) {
		generateConfig()
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSetup() {
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	orchestrator, err := setup.NewOrchestrator(cfg)
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}

	ctx := context.Background()
	if err := orchestrator.Setup(ctx); err != nil {
		log.Fatalf("Setup failed: %v", err)
	}
}

func validateConfig() {
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	fmt.Println("Configuration is valid!")
	fmt.Println("\nParsed configuration:")

	data, err := yaml.Marshal(cfg)
	if err != nil {
		log.Fatalf("Failed to marshal config: %v", err)
	}

	fmt.Print(string(data))
}

func generateConfig() {
	exampleConfig := `# TDX-Init Configuration File
# This configuration defines SSH key management, encryption keys, and disk setup

# SSH Configuration
ssh:
  # Strategy for obtaining SSH keys
  strategy: "webserver"  # Currently only 'webserver' is supported
  
  # Strategy-specific configuration
  strategy_config:
    # For webserver strategy: the address to listen on
    server_url: "0.0.0.0:8080"
  
  # SSH directory where authorized_keys will be created
  dir: "/root/.ssh"
  
  # Path to store the SSH key separately (optional)
  key_path: "/etc/root_key"
  
  # Store SSH key in LUKS token of specified disk (optional)
  # This allows the key to persist across reboots
  store_at: "disk_persistent"

# Encryption Key Configuration
keys:
  # Define one or more encryption keys
  key_persistent:
    # Strategy for key generation/retrieval
    strategy: "random"  # Options: 'random', 'pipe'
    
    # For 'pipe' strategy, specify the pipe path:
    # strategy_config:
    #   pipe_path: "/tmp/passphrase"
    
    # Store key in TPM if available
    tpm: true

# Disk Configuration
disks:
  # Define one or more disks to manage
  disk_persistent:
    # Strategy for finding the disk
    strategy: "largest"  # Options: 'largest', 'pathglob'
    
    # For 'pathglob' strategy, specify the pattern:
    # strategy_config:
    #   path_glob: "/dev/sd*"
    
    # When to format the disk
    # - 'always': Format on every run (DESTRUCTIVE!)
    # - 'on_initialize': Format only if not already initialized (default)
    # - 'never': Never format, only mount existing filesystems
    format: "on_initialize"
    
    # Encryption key to use (references a key from 'keys' section)
    # Leave empty for unencrypted disk
    encryption_key: "key_persistent"
    
    # Where to mount the disk
    mount_at: "/persistent"

  # Example of an additional unencrypted disk:
  # disk_data:
  #   strategy: "pathglob"
  #   strategy_config:
  #     path_glob: "/dev/nvme*"
  #   format: "on_initialize"
  #   mount_at: "/data"
`

	filename := "config.example.yaml"
	if err := os.WriteFile(filename, []byte(exampleConfig), 0644); err != nil {
		log.Fatalf("Failed to write example config: %v", err)
	}

	fmt.Printf("Example configuration written to %s\n", filename)
}
