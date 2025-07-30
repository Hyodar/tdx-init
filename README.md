# tdx-init

A configurable CLI tool for secure disk encryption and SSH key management in TDX (Trusted Domain Extensions) environments. Provides flexible strategies for key initialization, passphrase generation, and disk selection.

## Features

- **Configurable Key Strategies**: Web server-based SSH key provisioning
- **Flexible Passphrase Handling**: Random generation or named pipe input
- **Disk Selection Options**: Largest available disk or path glob matching
- **Secure SSH Configuration**: Automatically configures SSH with security restrictions
- **LUKS2 Integration**: Stores SSH keys in LUKS2 headers for persistence

## Installation

```bash
go build -o tdx-init ./cmd
```

## Usage

### Quick Setup

Run the complete setup process with default configuration:

```bash
tdx-init setup
```

### Configuration Options

View current configuration:

```bash
tdx-init config
```

### Custom Configuration

Configure key strategy:
```bash
tdx-init setup --key-strategy.type webserver --key-strategy.server-url :8080
```

Configure passphrase strategy:
```bash
tdx-init setup --passphrase-strategy.type random
# or
tdx-init setup --passphrase-strategy.type namedpipe --passphrase-strategy.pipe-path /tmp/passphrase
```

Configure disk strategy:
```bash
tdx-init setup --disk-strategy.type largest
# or
tdx-init setup --disk-strategy.type pathglob --disk-strategy.path-glob "/dev/sd*"
```

### Global Options

```bash
tdx-init setup \
  --ssh-dir /root/.ssh \
  --key-file /etc/root_key \
  --mount-point /persistent \
  --mapper-name cryptdisk \
  --mapper-device /dev/mapper/cryptdisk
```

## Strategies

### Key Strategies

- **webserver**: Waits for SSH key via HTTP server or extracts from existing LUKS header

### Passphrase Strategies

- **random**: Generates a secure random passphrase
- **namedpipe**: Reads passphrase from a named pipe

### Disk Strategies

- **largest**: Selects the largest available disk
- **pathglob**: Selects disk matching a path pattern

## Security Features

- SSH keys are stored with security restrictions (`no-port-forwarding,no-agent-forwarding,no-X11-forwarding`)
- Encrypted disk setup with LUKS2
- Secure file permissions (0600/0700)
- SSH key persistence in LUKS2 token metadata

## Requirements

- Go 1.22.1+
- cryptsetup (for LUKS operations)
- Root privileges (for disk and SSH operations)
