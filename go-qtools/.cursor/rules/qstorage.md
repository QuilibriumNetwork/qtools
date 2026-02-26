# QStorage / S3-Compatible Storage

## Repository Import Namespace

All code references in this guide should use `github.com/quilibrium/qtools/go-qtools/...`.

## Default Configuration

Quilibrium uses its own S3-compatible storage service called **QStorage**. All S3 operations in qtools should use these defaults unless explicitly overridden by the user:

- **Region**: `q-world-1`
- **Endpoint URL**: `https://qstorage.quilibrium.com`

These defaults are defined in `internal/s3/client.go` as constants:
```go
const (
    DefaultRegion      = "q-world-1"
    DefaultEndpointURL = "https://qstorage.quilibrium.com"
)
```

## Bucket Layout

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

### S3 <-> Local Path Mapping (1:1)

| S3 Path | Local Path |
|---------|-----------|
| `<prefix>/config.yml` | `$QUIL_NODE_PATH/.config/config.yml` |
| `<prefix>/keys.yml` | `$QUIL_NODE_PATH/.config/keys.yml` |
| `<prefix>/store/` | `$QUIL_NODE_PATH/.config/store/` |
| `<prefix>/worker-store/<id>/` | `$QUIL_NODE_PATH/.config/worker-store/<id>/` |

Directory constants are defined in `internal/s3/client.go`:
```go
const (
    BucketDirStore       = "store"        // master store data
    BucketDirWorkerStore = "worker-store" // worker-store/<id>/...
)
```

Use `client.GetObjectKey("config.yml")` for config files and `client.GetObjectKey(s3client.BucketDirStore)` for store directories -- this automatically prepends the user's prefix if set.

## Commands

Node backup/restore is handled by `qtools node backup` and `qtools node restore`:

- `qtools node backup` - Upload everything (config + master store + all workers)
- `qtools node backup --config-files` - Upload only config.yml + keys.yml
- `qtools node backup --master` - Upload only the master store
- `qtools node backup --worker 1,2,3` - Upload only specific worker stores
- `qtools node backup --master --worker 1,2` - Upload master + specific workers
- `qtools node restore` - Download everything (config + master store + all workers)
- `qtools node restore --config-files` - Download only config.yml + keys.yml
- `qtools node restore --config-files --keys-only` - Migrate key data only
- `qtools node restore --master` - Download only the master store
- `qtools node restore --worker 1,2,3` - Download only specific worker stores

The `--keys-only` flag is a modifier on `--config-files` that downloads to temp, creates a fresh default config if needed, and migrates only key data (keys.yml, encryptionKey, peerPrivKey) into the local config.

The `--master` and `--worker` flags scope store operations. When neither is provided, all stores (master + all discovered workers) are backed up or restored.

## S3 Client Usage

Always use the reusable `internal/s3` package for all S3/QStorage operations. Do NOT create AWS SDK clients directly in command handlers.

### Creating a Client

```go
import s3client "github.com/quilibrium/qtools/go-qtools/internal/s3"

client, err := s3client.NewClient(s3client.ClientConfig{
    AccessKeyID: "...",
    AccessKey:   "...",
    AccountID:   "...",
    Region:      "",  // Falls back to DefaultRegion
    EndpointURL: "",  // Falls back to DefaultEndpointURL
    Bucket:      "my-bucket",
    Prefix:      "",  // Optional root prefix
})
```

### Building Object Keys

Use the variadic `GetObjectKey` to build keys with automatic prefix handling:
```go
// Config files (flat at root):
// With prefix="" -> "config.yml"
// With prefix="mynode" -> "mynode/config.yml"
key := client.GetObjectKey("config.yml")

// Master store directory:
// With prefix="" -> "store"
// With prefix="mynode" -> "mynode/store"
key := client.GetObjectKey(s3client.BucketDirStore)

// Worker store directory:
// With prefix="" -> "worker-store/1"
// With prefix="mynode" -> "mynode/worker-store/1"
key := client.GetObjectKey(s3client.BucketDirWorkerStore, "1")
```

### General S3 Operations

```go
client.UploadFile(ctx, localPath, s3Key)
client.DownloadFile(ctx, s3Key, localPath)
client.ListObjects(ctx, prefix)
client.DeleteObject(ctx, s3Key)
```

## Credential Storage

QStorage credentials are stored in the qtools config under the `qstorage` section:

```yaml
qstorage:
  access_key_id: ""
  access_key: ""
  account_id: ""
  bucket: ""
  region: "q-world-1"
  endpoint_url: "https://qstorage.quilibrium.com"
  prefix: ""
```

Credentials are stored in **plain text**. Users are informed of this when prompted to save.

## Config Type

The `QStorageConfig` struct is defined in `internal/config/config.go`:
```go
type QStorageConfig struct {
    AccessKeyID string `yaml:"access_key_id,omitempty"`
    AccessKey   string `yaml:"access_key,omitempty"`
    AccountID   string `yaml:"account_id,omitempty"`
    Bucket      string `yaml:"bucket,omitempty"`
    Region      string `yaml:"region,omitempty"`
    EndpointURL string `yaml:"endpoint_url,omitempty"`
    Prefix      string `yaml:"prefix,omitempty"`
}
```

## Migration

QStorage config was added in config version **1.4** (migration from 1.3 -> 1.4 in `internal/config/migrations.go`).

## Important Notes

- **Never hardcode** AWS/S3 region or endpoint values. Always use the constants from `internal/s3/client.go` or read from config.
- The S3 client uses **path-style URLs** (`UsePathStyle = true`) which is required for most S3-compatible services.
- The `internal/s3` package is designed to be used across the project (node backup/restore, future qtools backup, etc.).
- Use `BucketDirStore` and `BucketDirWorkerStore` constants for directory names -- never hardcode these strings.
- Config files (`config.yml`, `keys.yml`) live at the bucket root (under prefix), not in a subdirectory.
