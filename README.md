# tdx-init

A utility for secure disk encryption and SSH configuration in TDX (Trusted Domain Extensions) VMs.
Currently designed specifically for Flashbots BoB VMs.

## Functionality

- Allows VM operator to provision a VM with a searcher's SSH key
- Stores searcher's SSH key directly in LUKS2 header
- Automatically sets up persistant encrypted disk once searcher has provided a LUKS2 passphrase

## Usage

```
tdx-init wait-for-key     # Wait for SSH key via HTTP or extract from existing LUKS header
tdx-init set-passphrase   # Set up or mount an encrypted disk with passphrase
```

- `tdx-init wait-for-key` should be called during the init process (before anything requiring peristant storage is mounted)
- `tdx-init set-passphrase` should be called after the SSH key has been established directly by the searcher through an authenticated channel
