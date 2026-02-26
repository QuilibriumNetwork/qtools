# Qtools Go Rewrite

This is the Go rewrite of qtools with TUI support, following the implementation plan in `.cursor/plans/go_tools_rewrite_with_tui.plan.md`.

## Status

### ✅ Completed

- **Phase 1: Foundation & Config Management**
  - ✅ Go module initialization
  - ✅ Directory structure
  - ✅ Config system with auto-migration (`internal/config/`)
    - `paths.go` - Path management
    - `config.go` - Config structs
    - `loader.go` - Config loading/saving
    - `migrations.go` - Migration system
    - `generator.go` - Default config generation

- **Phase 2: Node Setup**
  - ✅ Node config types (`internal/node/config_types.go`)
  - ✅ Node config manager (`internal/node/config.go`)
  - ✅ Config operations (`internal/node/config_operations.go`)
  - ✅ Mode detection (`internal/node/mode.go`)
  - ✅ Node setup (`internal/node/setup.go`)
  - ✅ Node commands (`internal/node/commands.go`)
  - ✅ Installation scaffolding (`internal/node/install.go`)

- **Phase 3: Service Management**
  - ✅ Service options (`internal/service/options.go`)
  - ✅ Service manager (`internal/service/manager.go`)
  - ✅ Platform detection (`internal/service/platform.go`)
  - ✅ Systemd integration (`internal/service/systemd.go`)
  - ✅ Launchd integration (`internal/service/launchd.go`)
  - ✅ Plist generation (`internal/service/plist.go`)
  - ✅ Worker management (`internal/service/workers.go`)

- **Phase 4: CLI Interface**
  - ✅ Basic CLI structure with Cobra (`cmd/qtools/main.go`)
  - ✅ Node commands (setup, mode, install)
  - ✅ Service commands (start, stop, restart, status)
  - ✅ TUI command integration

- **Phase 5: TUI Implementation**
  - ✅ Main TUI app (`internal/tui/app.go`)
  - ✅ Node setup view (`internal/tui/views/node_setup.go`)
  - ✅ Service control view (`internal/tui/views/service_control.go`)
  - ✅ Status view (`internal/tui/views/status.go`)
  - ✅ Log view (`internal/tui/views/log_view.go`)
  - ✅ Components (menu, core input)
  - ✅ Log filtering (`internal/log/filters.go`, `internal/log/viewer.go`)

- **Phase 6: Desktop Integration (Stubs)**
  - ✅ Messaging stub (`internal/messaging/stub.go`)
  - ✅ Node client (`internal/client/node_client.go`)
    - Binary command support (works when node is stopped)
    - gRPC support stub (for when node is running)
    - Hybrid approach (tries gRPC first, falls back to binary)

### 🚧 Future Enhancements

- **Phase 2: Node Setup**
  - ⏳ Complete `install.go` implementation (download binaries, create users/groups, etc.)

- **Phase 4: CLI Interface**
  - ⏳ Implement actual command handlers (currently stubs)
  - ⏳ Log commands
  - ⏳ Config commands

- **Phase 6: Desktop Integration**
  - ⏳ Implement actual Quilibrium Messaging integration
  - ⏳ Full gRPC client implementation
  - ⏳ Desktop app SDK/examples

## Building

```bash
cd go-tools
CGO_ENABLED=0 GOOS=linux GOARCH=$(go env GOARCH) go build -o qtools ./cmd/qtools
```

## Upload Dev Builds To QStorage

Use the helper script to upload local binaries to QStorage and rotate bucket paths:

- Existing objects in `<prefix>/current/` are moved to `<prefix>/old/<timestamp>/`
- New artifacts are uploaded to `<prefix>/current/`

```bash
cd go-qtools
task build:docker:all
task upload:qstorage
```

You can also run the script directly:

```bash
cd go-qtools
./scripts/upload-qstorage-build.sh \
  --bucket <bucket-name> \
  --access-key-id <access-key-id> \
  --access-key <access-key>
```

Defaults:
- Region: `q-world-1`
- Endpoint: `https://qstorage.quilibrium.com`
- Prefix: `qtools/dev-builds`
- Artifacts: `dist/qtools` and `dist/qtools-arm64`

## Running

```bash
# CLI
./qtools --help
./qtools node setup --help
./qtools service start --help

# TUI
./qtools tui
```

## Functional Testing (Incus)

You can run distro-level functional testing against fresh binary builds with Incus:

```bash
cd go-qtools
task test:functional:incus
```

What this does:
- Builds a fresh `qtools` binary (`dist/qtools-functional`)
- Launches Incus containers for Ubuntu LTS and Debian stable
- Optionally runs a lightweight distro (Alpine) in best-effort mode
- Runs `qtools node install` and validates installation outcomes:
  - node directory created
  - node binary installed and executable
  - `/usr/local/bin/quilibrium-node` symlink created and on `PATH`
  - qtools config generated

Useful environment variables:
- `SKIP_BUILD=1` to reuse an existing binary
- `BINARY_PATH=/path/to/qtools` to test a specific binary
- `INCLUDE_LIGHTWEIGHT=0` to skip lightweight distro checks
- `KEEP_CONTAINERS=1` to preserve containers for debugging

## Shell Completion

Qtools supports shell completion for bash, zsh, fish, and PowerShell.

### Installation

**Automatic installation (recommended):**

Simply run `qtools completion` without arguments. Qtools will:
- Auto-detect your shell
- Prompt you if detection fails
- Install completions permanently

```bash
# Auto-detect and install
qtools completion

# Or specify shell explicitly
qtools completion bash
qtools completion zsh
qtools completion fish
```

**Generate completion script (for manual installation):**

If you prefer to install manually, use the `--generate` flag:

```bash
# Generate to stdout
qtools completion bash --generate > ~/.local/share/bash-completion/completions/qtools

# Or use the installation script
./scripts/install-completion.sh bash
```

**PowerShell:**

PowerShell completion requires manual setup:
```powershell
qtools completion powershell --generate | Out-String | Invoke-Expression
# Or add to profile:
qtools completion powershell --generate | Out-String | Add-Content $PROFILE
```

After installation, restart your shell or source the completion file as instructed.

## Architecture

The project follows the structure defined in the plan:

```
go-tools/
├── cmd/qtools/          # CLI entry point
├── internal/
│   ├── config/         # Config management
│   ├── node/           # Node setup and management
│   ├── s3/             # S3/QStorage client library
│   ├── service/        # Service management
│   ├── log/            # Log viewing and filtering
│   ├── tui/            # TUI implementation
│   ├── messaging/      # Desktop integration stub
│   └── client/         # Node client library
```

## Key Features

- **Dynamic Config System**: Auto-migrating config with programmatic defaults
- **Manual Mode Default**: Opinionated default for better reliability (each worker as separate service)
- **Cross-platform**: Linux (systemd) and macOS (launchd) support
- **TUI Support**: Full Bubble Tea-based TUI interface
- **Service Management**: Complete service control for master and workers
- **Log Viewing**: Real-time log tailing with filtering support
- **QStorage Integration**: Fetch credentials and backup to Quilibrium's S3-compatible storage
- **Key Migration**: Go-native implementation of key data migration from config files

## Command Structure

Commands are organized into separate branches:

- **`qtools node`** - All node-related commands:
  - `node setup` - Setup node
  - `node install` - Complete installation
  - `node download` - Download node binary
  - `node update` - Update node binary
  - `node config` - Node configuration management
  - `node backup` - Backup node data to QStorage (S3)
  - `node restore` - Restore node data from QStorage (S3)
  - `node info` - Get node information
  - `node peer-id` - Get peer ID
  - `node balance` - Get balance
  - `node seniority` - Get seniority
  - `node worker-count` - Get worker count

- **`qtools qclient`** - All qclient-related commands:
  - `qclient download` - Download qclient binary
  - (More qclient commands coming: transfer, merge, split, coins, account, etc.)

- **Other top-level commands**:
  - `qtools service` - Service management (start, stop, restart, status)
  - `qtools config` - Qtools configuration management
  - `qtools toggle` - Toggle settings (auto-updates, etc.)
  - `qtools util` - Utility commands (public-ip, etc.)
  - `qtools logs` - Log viewing and management
  - `qtools backup` - Backup and restore (local/peer)
  - `qtools diagnostics` - Diagnostic commands
  - `qtools update` - Update commands
  - `qtools completion` - Shell completion
  - `qtools tui` - Launch TUI mode

## QStorage / S3 Integration

Qtools integrates with Quilibrium's **QStorage** service (S3-compatible) for backing up and restoring node data.

### Prerequisites

Before using the backup/restore commands, you need to set up a QStorage (or S3-compatible) account with the following:

1. **Create a user** with programmatic access (access key ID + access key).
2. **Create a bucket** to store your node backups.
3. **Attach an S3 read/write policy** to the user, scoped to the bucket. At minimum the user needs permissions for `s3:GetObject`, `s3:PutObject`, `s3:DeleteObject`, and `s3:ListBucket` on the target bucket.

Example S3 policy (adjust `your-bucket-name` as needed):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject"
      ],
      "Resource": "arn:aws:s3:::your-bucket-name/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:ListBucket"
      ],
      "Resource": "arn:aws:s3:::your-bucket-name"
    }
  ]
}
```

> **Tip**: Create a dedicated user per node rather than reusing credentials across nodes. This limits the blast radius if a key is compromised.

### Default Connection Settings

| Setting      | Default Value                        |
|-------------|--------------------------------------|
| Region      | `q-world-1`                          |
| Endpoint    | `https://qstorage.quilibrium.com`    |

These defaults can be overridden per-command via flags or saved in the qtools config.

### Bucket Layout

The S3 bucket mirrors the local `$QUIL_NODE_PATH/.config/` directory structure directly:

```
<bucket>/
├── <prefix>/                      # Optional user-provided root prefix
│   ├── config.yml                 # Node configuration
│   ├── keys.yml                   # Node keys
│   ├── store/                     # Master store data
│   └── worker-store/              # Worker store data
│       ├── 1/                     # Worker 1
│       ├── 2/                     # Worker 2
│       └── ...
```

#### S3 <-> Local Path Mapping (1:1)

| S3 Path | Local Path |
|---------|-----------|
| `<prefix>/config.yml` | `$QUIL_NODE_PATH/.config/config.yml` |
| `<prefix>/keys.yml` | `$QUIL_NODE_PATH/.config/keys.yml` |
| `<prefix>/store/` | `$QUIL_NODE_PATH/.config/store/` |
| `<prefix>/worker-store/<id>/` | `$QUIL_NODE_PATH/.config/worker-store/<id>/` |

### Backing Up Node Data

Use `qtools node backup` to upload node data to a QStorage bucket.

```bash
# Backup everything (interactive, prompts for credentials)
qtools node backup

# Backup config files only
qtools node backup --config-files --bucket my-bucket

# Backup master store only
qtools node backup --master --bucket my-bucket

# Backup specific workers only
qtools node backup --worker 1,2,3 --bucket my-bucket

# Backup master + specific workers
qtools node backup --master --worker 1,2 --bucket my-bucket

# Provide all credentials via flags
qtools node backup \
  --access-key-id <key-id> \
  --access-key <key> \
  --account-id <account-id> \
  --bucket my-bucket
```

### Restoring Node Data

Use `qtools node restore` to download node data from a QStorage bucket.

```bash
# Restore everything (interactive)
qtools node restore

# Restore config files only
qtools node restore --config-files --bucket my-bucket

# Restore keys only (migrate key data into existing/new config)
qtools node restore --config-files --keys-only --bucket my-bucket

# Restore master store only
qtools node restore --master --bucket my-bucket

# Restore specific workers only
qtools node restore --worker 1,2,3 --bucket my-bucket

# Restore master + specific workers
qtools node restore --master --worker 1,2 --bucket my-bucket

# Provide all credentials via flags
qtools node restore \
  --access-key-id <key-id> \
  --access-key <key> \
  --account-id <account-id> \
  --bucket my-bucket

# Override default region and endpoint
qtools node restore --bucket my-bucket \
  --region us-east-1 \
  --endpoint-url https://s3.amazonaws.com
```

**`--config-files`**: Only operate on config files (`config.yml` + `keys.yml`) rather than the full node data.

**`--keys-only`** (modifier for `--config-files`): Instead of replacing files directly, downloads to a temp directory, creates a fresh default node config (if needed), and migrates only the key data (`keys.yml`, `encryptionKey`, `peerPrivKey`) into the local config. Cleans up temp files afterward.

**`--master`**: Only operate on the master store. Can be combined with `--worker`.

**`--worker <ids>`**: Only operate on specific worker store(s). Accepts a comma-separated list of worker IDs (e.g., `1,2,3`). Can be combined with `--master`.

### Shared Flags

| Flag               | Description                                    |
|-------------------|------------------------------------------------|
| `--access-key-id` | QStorage/S3 access key ID                      |
| `--access-key`    | QStorage/S3 access key                         |
| `--account-id`    | QStorage/S3 account ID                         |
| `--bucket`        | S3 bucket name                                 |
| `--region`        | S3 region (default: `q-world-1`)               |
| `--endpoint-url`  | S3 endpoint URL (default: QStorage endpoint)   |
| `--prefix`        | Optional root prefix within the bucket         |
| `--config-files`  | Only operate on config files                   |
| `--keys-only`     | Migrate key data only (restore, with `--config-files`) |
| `--master`        | Only operate on the master store               |
| `--worker`        | Only operate on specific worker stores (comma-separated IDs) |

### Saving Credentials

After backup/restore, you'll be prompted to save credentials for future use. Credentials are stored in **plain text** in the qtools config file under the `qstorage` section:

```yaml
qstorage:
  access_key_id: "your-key-id"
  access_key: "your-access-key"
  account_id: "your-account-id"
  bucket: "your-bucket"
  region: "q-world-1"
  endpoint_url: "https://qstorage.quilibrium.com"
  prefix: ""
```

## Status

✅ **All core phases complete!** The foundation is fully implemented:
- Config system with auto-migration
- Node setup and management
- Service management (systemd/launchd)
- CLI interface
- TUI interface
- Desktop integration stubs

## Next Steps

1. Wire up CLI command handlers - Connect service management to CLI commands
2. Complete install.go implementation - Download binaries, user/group creation
3. Implement actual Quilibrium Messaging integration - Replace stubs with real implementation
4. Add comprehensive testing - Unit tests, integration tests
5. Add documentation - User guide, API documentation
