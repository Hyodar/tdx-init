# tdx-init

A configurable CLI tool for secure disk encryption and SSH key management in TDX (Trusted Domain Extensions) environments. Provides flexible strategies for key initialization, passphrase generation, and disk selection with TPM support.

## Features

- **YAML Configuration**: Flexible configuration system for all components
- **TPM Integration**: Hardware-based key storage using TPM 2.0
- **Multiple Key Strategies**: 
  - Random generation with hardware RNG support
  - Named pipe input for external key providers
- **Flexible Disk Selection**:
  - Largest available disk
  - Path glob pattern matching
- **Format Strategies**:
  - `always`: Format on every run
  - `on_initialize`: Format only on first setup (default)
  - `never`: Never format, only mount existing
- **SSH Key Persistence**: Store SSH keys in LUKS headers for persistence across reboots
- **Security Features**:
  - LUKS2 encryption with token support
  - SSH restrictions (no-port-forwarding, no-agent-forwarding, no-X11-forwarding)
  - Secure file permissions

## Installation

```bash
go build -o tdx-init ./cmd
```

## Usage

### Quick Start

1. Generate an example configuration file:
```bash
./tdx-init generate-config
```

2. Customize the configuration (see `config.example.yaml`)

3. Validate your configuration:
```bash
./tdx-init validate config.yaml
```

4. Run the setup:
```bash
./tdx-init setup config.yaml
```

## Configuration

The tool uses YAML configuration files. Here's a complete example:

```yaml
# SSH Configuration
ssh:
  strategy: "webserver"        # Currently only 'webserver' is supported
  strategy_config:
    server_url: "0.0.0.0:8080" # Address to listen for SSH keys
  dir: "/root/.ssh"            # SSH directory
  key_path: "/etc/root_key"    # Optional: store key separately
  store_at: "disk_persistent"  # Optional: store in LUKS token

# Encryption Keys
keys:
  key_persistent:
    strategy: "random"         # Options: 'random', 'pipe'
    tpm: true                  # Store in TPM if available
    
  # Example pipe strategy:
  # key_external:
  #   strategy: "pipe"
  #   strategy_config:
  #     pipe_path: "/tmp/passphrase"
  #   tpm: false

# Disk Configuration
disks:
  disk_persistent:
    strategy: "largest"        # Options: 'largest', 'pathglob'
    format: "on_initialize"    # Options: 'always', 'on_initialize', 'never'
    encryption_key: "key_persistent"  # Reference to key in 'keys' section
    mount_at: "/persistent"
    
  # Example pathglob strategy:
  # disk_data:
  #   strategy: "pathglob"
  #   strategy_config:
  #     path_glob: "/dev/nvme*"
  #   format: "on_initialize"
  #   mount_at: "/data"
```

## Architecture

### Component Organization

```
pkg/
├── config/          # Configuration parsing and validation
├── keys/            # Key management strategies
│   ├── random.go    # Random key generation with HW RNG support
│   └── pipe.go      # Named pipe key input
├── disks/           # Disk management
│   ├── largest.go   # Find largest available disk
│   ├── pathglob.go  # Match disks by pattern
│   ├── luks.go      # LUKS operations
│   └── filesystem.go # Filesystem operations
├── ssh/             # SSH key management
│   └── webserver.go # HTTP server for key reception
├── tpm/             # TPM 2.0 integration
└── setup/           # Orchestration layer
```

### How It Works

1. **Initial Setup**:
   - Finds disk based on configured strategy
   - Generates or receives encryption key
   - Formats disk with LUKS2 if needed
   - Stores initialization token in LUKS header
   - Waits for SSH key via HTTP POST
   - Stores SSH key in LUKS token (if configured)

2. **Subsequent Boots**:
   - Detects existing LUKS container
   - Retrieves SSH key from LUKS token (if stored)
   - Retrieves encryption key from TPM (if available)
   - Mounts encrypted filesystem
   - Configures SSH access

### LUKS Token Usage

- **Token Slot 1**: SSH public key storage
- **Token Slot 2**: Initialization state tracking

### TPM Integration

When TPM is available and enabled:
- Keys are stored in TPM NV index (default: 0x1500016)
- Automatic key retrieval on subsequent boots
- Fallback to non-TPM operation if unavailable

## Security Considerations

- **No Private Keys**: Only public SSH keys are handled
- **Passphrase Security**: Encryption passphrases never stored on disk (only in TPM)
- **SSH Restrictions**: Automatic security restrictions on SSH keys
- **Secure Permissions**: Files created with appropriate permissions (0600/0700)

## Requirements

- Go 1.22.1+
- Linux with `/proc/partitions` support
- cryptsetup (for LUKS operations)
- TPM 2.0 tools (optional, for TPM support)
- Root privileges

## Development

The project uses a clean architecture with:
- **Strategy Pattern**: For different implementations of key/disk/ssh strategies
- **Factory Pattern**: For creating appropriate strategy instances
- **Interface Segregation**: Clear interfaces between components
- **Dependency Injection**: Managers receive their dependencies explicitly

## License

See LICENSE file for details.