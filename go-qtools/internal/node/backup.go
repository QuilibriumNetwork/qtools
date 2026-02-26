package node

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/quilibrium/qtools/go-qtools/internal/config"
	s3client "github.com/quilibrium/qtools/go-qtools/internal/s3"
)

// BackupRestoreOptions holds the options for node backup and restore commands
type BackupRestoreOptions struct {
	// S3/QStorage credentials (can be provided via flags or prompted)
	AccessKeyID string
	AccessKey   string
	AccountID   string

	// S3/QStorage connection settings
	Bucket      string
	Region      string
	EndpointURL string
	Prefix      string

	// Scope flags
	ConfigFiles bool  // If true, only operate on config files (config.yml + keys.yml)
	KeysOnly    bool  // Modifier on ConfigFiles: migrate key data instead of replacing files
	Master      bool  // If true, only operate on master store
	Workers     []int // If non-empty, only operate on these worker stores
}

// ParseWorkerIDs parses a comma-separated string of worker IDs into a slice of ints.
// Example: "1,2,3" -> []int{1, 2, 3}
func ParseWorkerIDs(csv string) ([]int, error) {
	if csv == "" {
		return nil, nil
	}
	parts := strings.Split(csv, ",")
	ids := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid worker ID %q: %w", p, err)
		}
		if id < 1 {
			return nil, fmt.Errorf("worker ID must be >= 1, got %d", id)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// hasStoreScope returns true if the user specified --master or --worker,
// meaning they want to operate on specific store components rather than everything.
func (o *BackupRestoreOptions) hasStoreScope() bool {
	return o.Master || len(o.Workers) > 0
}

// BackupNode uploads node data to an S3-compatible bucket
func BackupNode(opts BackupRestoreOptions, cfg *config.Config) error {
	reader := bufio.NewReader(os.Stdin)

	// Resolve credentials: flags > saved config > prompt
	opts = resolveCredentials(opts, cfg, reader)

	// Validate required fields
	if err := validateS3Options(opts); err != nil {
		return err
	}

	// Create S3 client
	client, err := createS3Client(opts)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// If --config-files is set, only backup config files
	if opts.ConfigFiles {
		return backupConfigFiles(ctx, client)
	}

	// If --master or --worker is set, only backup the specified store(s)
	if opts.hasStoreScope() {
		return backupStoreScoped(ctx, client, opts)
	}

	// Full backup: config files + master store + all worker stores
	fmt.Println("Backing up config files...")
	if err := backupConfigFiles(ctx, client); err != nil {
		return err
	}

	fmt.Println("Backing up master store...")
	if err := backupMasterStore(ctx, client); err != nil {
		return err
	}

	fmt.Println("Backing up worker stores...")
	if err := backupAllWorkerStores(ctx, client); err != nil {
		return err
	}

	fmt.Println("Backup completed successfully.")
	return nil
}

// RestoreNode downloads node data from an S3-compatible bucket
func RestoreNode(opts BackupRestoreOptions, cfg *config.Config) error {
	reader := bufio.NewReader(os.Stdin)

	// Resolve credentials: flags > saved config > prompt
	opts = resolveCredentials(opts, cfg, reader)

	// Validate required fields
	if err := validateS3Options(opts); err != nil {
		return err
	}

	// Create S3 client
	client, err := createS3Client(opts)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// If --config-files is set, only restore config files
	if opts.ConfigFiles {
		if opts.KeysOnly {
			return restoreKeysOnly(ctx, client, cfg, reader)
		}
		return restoreConfigFiles(ctx, client, reader)
	}

	// If --master or --worker is set, only restore the specified store(s)
	if opts.hasStoreScope() {
		return restoreStoreScoped(ctx, client, opts, reader)
	}

	// Full restore: config files + master store + all worker stores
	fmt.Println("Restoring config files...")
	if err := restoreConfigFiles(ctx, client, reader); err != nil {
		return err
	}

	fmt.Println("Restoring master store...")
	if err := restoreMasterStore(ctx, client, reader); err != nil {
		return err
	}

	fmt.Println("Restoring worker stores...")
	if err := restoreAllWorkerStores(ctx, client, reader); err != nil {
		return err
	}

	fmt.Println("Restore completed successfully.")
	return nil
}

// validateS3Options validates that all required S3 options are present
func validateS3Options(opts BackupRestoreOptions) error {
	if opts.AccessKeyID == "" || opts.AccessKey == "" || opts.AccountID == "" {
		return fmt.Errorf("access key ID, access key, and account ID are all required")
	}
	if opts.Bucket == "" {
		return fmt.Errorf("bucket name is required (use --bucket or save it to config)")
	}
	return nil
}

// createS3Client creates an S3 client from the resolved options
func createS3Client(opts BackupRestoreOptions) (*s3client.Client, error) {
	region := opts.Region
	if region == "" {
		region = s3client.DefaultRegion
	}
	endpointURL := opts.EndpointURL
	if endpointURL == "" {
		endpointURL = s3client.DefaultEndpointURL
	}

	client, err := s3client.NewClient(s3client.ClientConfig{
		AccessKeyID: opts.AccessKeyID,
		AccessKey:   opts.AccessKey,
		AccountID:   opts.AccountID,
		Region:      region,
		EndpointURL: endpointURL,
		Bucket:      opts.Bucket,
		Prefix:      opts.Prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}
	return client, nil
}

// resolveCredentials resolves credentials from flags, saved config, or interactive prompts
func resolveCredentials(opts BackupRestoreOptions, cfg *config.Config, reader *bufio.Reader) BackupRestoreOptions {
	if cfg != nil && cfg.QStorage != nil {
		if opts.AccessKeyID == "" {
			opts.AccessKeyID = cfg.QStorage.AccessKeyID
		}
		if opts.AccessKey == "" {
			opts.AccessKey = cfg.QStorage.AccessKey
		}
		if opts.AccountID == "" {
			opts.AccountID = cfg.QStorage.AccountID
		}
		if opts.Bucket == "" {
			opts.Bucket = cfg.QStorage.Bucket
		}
		if opts.Region == "" && cfg.QStorage.Region != "" {
			opts.Region = cfg.QStorage.Region
		}
		if opts.EndpointURL == "" && cfg.QStorage.EndpointURL != "" {
			opts.EndpointURL = cfg.QStorage.EndpointURL
		}
		if opts.Prefix == "" && cfg.QStorage.Prefix != "" {
			opts.Prefix = cfg.QStorage.Prefix
		}
	}

	if opts.AccessKeyID == "" {
		opts.AccessKeyID = promptInput(reader, "Access Key ID")
	}
	if opts.AccessKey == "" {
		opts.AccessKey = promptInput(reader, "Access Key")
	}
	if opts.AccountID == "" {
		opts.AccountID = promptInput(reader, "Account ID")
	}
	if opts.Bucket == "" {
		opts.Bucket = promptInput(reader, "Bucket name")
	}

	return opts
}

// promptInput prompts the user for input and returns the trimmed value
func promptInput(reader *bufio.Reader, label string) string {
	fmt.Printf("%s: ", label)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// promptYesNo prompts the user for a yes/no answer and returns true for yes
func promptYesNo(reader *bufio.Reader, question string) bool {
	fmt.Printf("%s [y/N]: ", question)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

// PromptSaveCredentials asks the user if they want to save credentials to the qtools config
func PromptSaveCredentials(opts BackupRestoreOptions, cfg *config.Config, reader *bufio.Reader) error {
	if cfg.QStorage != nil &&
		cfg.QStorage.AccessKeyID == opts.AccessKeyID &&
		cfg.QStorage.AccessKey == opts.AccessKey &&
		cfg.QStorage.AccountID == opts.AccountID &&
		cfg.QStorage.Bucket == opts.Bucket {
		return nil
	}

	fmt.Println()
	fmt.Println("NOTE: Credentials will be saved in plain text in the qtools config file.")
	if !promptYesNo(reader, "Save these QStorage credentials for future use?") {
		fmt.Println("Credentials not saved.")
		return nil
	}

	if cfg.QStorage == nil {
		cfg.QStorage = &config.QStorageConfig{}
	}
	cfg.QStorage.AccessKeyID = opts.AccessKeyID
	cfg.QStorage.AccessKey = opts.AccessKey
	cfg.QStorage.AccountID = opts.AccountID
	cfg.QStorage.Bucket = opts.Bucket
	if opts.Region != "" {
		cfg.QStorage.Region = opts.Region
	}
	if opts.EndpointURL != "" {
		cfg.QStorage.EndpointURL = opts.EndpointURL
	}
	if opts.Prefix != "" {
		cfg.QStorage.Prefix = opts.Prefix
	}

	configPath := config.GetConfigPath()
	if err := config.SaveConfig(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save credentials to config: %w", err)
	}

	fmt.Println("Credentials saved to qtools config.")
	return nil
}

// ============================================================================
// Backup operations
// ============================================================================

// backupConfigFiles uploads config.yml and keys.yml to <prefix>/config.yml and <prefix>/keys.yml
func backupConfigFiles(ctx context.Context, client *s3client.Client) error {
	nodeConfigDir := filepath.Join(config.GetNodePath(), ".config")

	configSrc := filepath.Join(nodeConfigDir, "config.yml")
	keysSrc := filepath.Join(nodeConfigDir, "keys.yml")

	if _, err := os.Stat(configSrc); err == nil {
		fmt.Println("Uploading config.yml...")
		key := client.GetObjectKey("config.yml")
		if err := client.UploadFile(ctx, configSrc, key); err != nil {
			return fmt.Errorf("failed to upload config.yml: %w", err)
		}
		fmt.Printf("  -> %s/%s\n", client.GetBucket(), key)
	} else {
		fmt.Println("Warning: config.yml not found, skipping.")
	}

	if _, err := os.Stat(keysSrc); err == nil {
		fmt.Println("Uploading keys.yml...")
		key := client.GetObjectKey("keys.yml")
		if err := client.UploadFile(ctx, keysSrc, key); err != nil {
			return fmt.Errorf("failed to upload keys.yml: %w", err)
		}
		fmt.Printf("  -> %s/%s\n", client.GetBucket(), key)
	} else {
		fmt.Println("Warning: keys.yml not found, skipping.")
	}

	fmt.Println("Config files backed up successfully.")
	return nil
}

// backupStoreScoped backs up only the stores specified by --master / --worker flags
func backupStoreScoped(ctx context.Context, client *s3client.Client, opts BackupRestoreOptions) error {
	if opts.Master {
		fmt.Println("Backing up master store...")
		if err := backupMasterStore(ctx, client); err != nil {
			return err
		}
	}
	for _, id := range opts.Workers {
		fmt.Printf("Backing up worker %d store...\n", id)
		if err := backupWorkerStore(ctx, client, id); err != nil {
			return err
		}
	}
	fmt.Println("Scoped backup completed successfully.")
	return nil
}

// backupMasterStore uploads the master store directory
// Local: $QUIL_NODE_PATH/.config/store/ -> S3: <prefix>/store/
func backupMasterStore(ctx context.Context, client *s3client.Client) error {
	localDir := getMasterStoreDir()
	s3Dir := s3client.BucketDirStore
	return uploadStoreDir(ctx, client, localDir, s3Dir, "master store")
}

// backupWorkerStore uploads a single worker store directory
// Local: $QUIL_NODE_PATH/.config/worker-store/<id>/ -> S3: <prefix>/worker-store/<id>/
func backupWorkerStore(ctx context.Context, client *s3client.Client, workerID int) error {
	localDir := getWorkerStoreDir(workerID)
	s3Dir := fmt.Sprintf("%s/%d", s3client.BucketDirWorkerStore, workerID)
	label := fmt.Sprintf("worker %d store", workerID)
	return uploadStoreDir(ctx, client, localDir, s3Dir, label)
}

// backupAllWorkerStores discovers and uploads all worker-store/<id> directories
func backupAllWorkerStores(ctx context.Context, client *s3client.Client) error {
	workerStoreBase := getWorkerStoreBase()

	entries, err := os.ReadDir(workerStoreBase)
	if os.IsNotExist(err) {
		fmt.Println("No worker-store directory found, skipping workers.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read worker-store directory: %w", err)
	}

	found := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Each subdirectory name is the worker ID (e.g., "1", "2")
		id, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // skip non-numeric directories
		}
		fmt.Printf("Backing up worker %d store...\n", id)
		if err := backupWorkerStore(ctx, client, id); err != nil {
			return err
		}
		found++
	}

	if found == 0 {
		fmt.Println("No worker stores found.")
	}
	return nil
}

// uploadStoreDir is a helper that uploads a local directory to <prefix>/<s3Dir>/
func uploadStoreDir(ctx context.Context, client *s3client.Client, localDir string, s3Dir string, label string) error {
	info, err := os.Stat(localDir)
	if os.IsNotExist(err) {
		fmt.Printf("  %s not found at %s, skipping.\n", label, localDir)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", label, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s path %s is not a directory", label, localDir)
	}

	s3Prefix := client.GetObjectKey(s3Dir)
	fmt.Printf("  Uploading %s -> %s/%s/...\n", localDir, client.GetBucket(), s3Prefix)

	count, err := client.UploadDirectory(ctx, localDir, s3Prefix)
	if err != nil {
		return fmt.Errorf("failed to upload %s: %w", label, err)
	}

	fmt.Printf("  Uploaded %d files.\n", count)
	return nil
}

// ============================================================================
// Restore operations
// ============================================================================

// restoreConfigFiles downloads config.yml and keys.yml from S3 to $QUIL_NODE_PATH/.config/
func restoreConfigFiles(ctx context.Context, client *s3client.Client, reader *bufio.Reader) error {
	nodeConfigDir := filepath.Join(config.GetNodePath(), ".config")

	if err := os.MkdirAll(nodeConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create node config directory: %w", err)
	}

	configDest := filepath.Join(nodeConfigDir, "config.yml")
	keysDest := filepath.Join(nodeConfigDir, "keys.yml")

	if _, err := os.Stat(configDest); err == nil {
		fmt.Printf("Config file already exists at %s\n", configDest)
		if !promptYesNo(reader, "Delete existing config and continue?") {
			fmt.Println("Operation cancelled.")
			return nil
		}
		if err := backupExistingConfig(nodeConfigDir); err != nil {
			return fmt.Errorf("failed to backup existing config: %w", err)
		}
	}

	if _, err := os.Stat(keysDest); err == nil {
		fmt.Printf("Keys file already exists at %s\n", keysDest)
		if !promptYesNo(reader, "Delete existing keys file and continue?") {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	fmt.Println("Downloading config.yml...")
	configKey := client.GetObjectKey("config.yml")
	if err := client.DownloadFile(ctx, configKey, configDest); err != nil {
		return fmt.Errorf("failed to download config.yml: %w", err)
	}
	fmt.Printf("  -> %s\n", configDest)

	fmt.Println("Downloading keys.yml...")
	keysKey := client.GetObjectKey("keys.yml")
	if err := client.DownloadFile(ctx, keysKey, keysDest); err != nil {
		return fmt.Errorf("failed to download keys.yml: %w", err)
	}
	fmt.Printf("  -> %s\n", keysDest)

	fmt.Println("Config files restored successfully.")
	return nil
}

// restoreKeysOnly downloads to temp, migrates key data into existing/fresh config
func restoreKeysOnly(ctx context.Context, client *s3client.Client, cfg *config.Config, reader *bufio.Reader) error {
	nodeConfigDir := filepath.Join(config.GetNodePath(), ".config")

	if err := os.MkdirAll(nodeConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create node config directory: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "qtools-restore-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpConfigPath := filepath.Join(tmpDir, "config.yml")
	tmpKeysPath := filepath.Join(tmpDir, "keys.yml")

	fmt.Println("Downloading config.yml to temp...")
	configKey := client.GetObjectKey("config.yml")
	if err := client.DownloadFile(ctx, configKey, tmpConfigPath); err != nil {
		return fmt.Errorf("failed to download config.yml: %w", err)
	}

	fmt.Println("Downloading keys.yml to temp...")
	keysKey := client.GetObjectKey("keys.yml")
	if err := client.DownloadFile(ctx, keysKey, tmpKeysPath); err != nil {
		return fmt.Errorf("failed to download keys.yml: %w", err)
	}

	localConfigPath := filepath.Join(nodeConfigDir, "config.yml")
	if _, err := os.Stat(localConfigPath); err == nil {
		fmt.Printf("Config file already exists at %s\n", localConfigPath)
		if !promptYesNo(reader, "Delete existing config and create a fresh default?") {
			fmt.Println("Keeping existing config, will migrate key data into it.")
		} else {
			if err := backupExistingConfig(nodeConfigDir); err != nil {
				return fmt.Errorf("failed to backup existing config: %w", err)
			}
			if err := os.Remove(localConfigPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove existing config: %w", err)
			}
			fmt.Println("Creating default node config...")
			if err := generateDefaultNodeConfig(localConfigPath); err != nil {
				return fmt.Errorf("failed to generate default node config: %w", err)
			}
		}
	} else {
		fmt.Println("Creating default node config...")
		if err := generateDefaultNodeConfig(localConfigPath); err != nil {
			return fmt.Errorf("failed to generate default node config: %w", err)
		}
	}

	fmt.Println("Migrating key data...")
	if err := MigrateKeyData(tmpDir, nodeConfigDir); err != nil {
		return fmt.Errorf("failed to migrate key data: %w", err)
	}

	fmt.Println("Key data migrated successfully.")
	return nil
}

// restoreStoreScoped restores only the stores specified by --master / --worker flags
func restoreStoreScoped(ctx context.Context, client *s3client.Client, opts BackupRestoreOptions, reader *bufio.Reader) error {
	if opts.Master {
		fmt.Println("Restoring master store...")
		if err := restoreMasterStore(ctx, client, reader); err != nil {
			return err
		}
	}
	for _, id := range opts.Workers {
		fmt.Printf("Restoring worker %d store...\n", id)
		if err := restoreWorkerStore(ctx, client, id, reader); err != nil {
			return err
		}
	}
	fmt.Println("Scoped restore completed successfully.")
	return nil
}

// restoreMasterStore downloads the master store
// S3: <prefix>/store/ -> Local: $QUIL_NODE_PATH/.config/store/
func restoreMasterStore(ctx context.Context, client *s3client.Client, reader *bufio.Reader) error {
	localDir := getMasterStoreDir()
	s3Dir := s3client.BucketDirStore
	return downloadStoreDir(ctx, client, s3Dir, localDir, "master store", reader)
}

// restoreWorkerStore downloads a single worker store
// S3: <prefix>/worker-store/<id>/ -> Local: $QUIL_NODE_PATH/.config/worker-store/<id>/
func restoreWorkerStore(ctx context.Context, client *s3client.Client, workerID int, reader *bufio.Reader) error {
	localDir := getWorkerStoreDir(workerID)
	s3Dir := fmt.Sprintf("%s/%d", s3client.BucketDirWorkerStore, workerID)
	label := fmt.Sprintf("worker %d store", workerID)
	return downloadStoreDir(ctx, client, s3Dir, localDir, label, reader)
}

// restoreAllWorkerStores discovers all worker-store/<id> prefixes in S3 and restores them
func restoreAllWorkerStores(ctx context.Context, client *s3client.Client, reader *bufio.Reader) error {
	// List objects under worker-store/ to discover worker directories
	wsPrefix := client.GetObjectKey(s3client.BucketDirWorkerStore) + "/"
	keys, err := client.ListObjects(ctx, wsPrefix)
	if err != nil {
		return fmt.Errorf("failed to list worker store objects: %w", err)
	}

	// Discover unique worker IDs from keys like "worker-store/1/...", "worker-store/2/..."
	workerIDs := discoverWorkerIDs(keys, wsPrefix)
	if len(workerIDs) == 0 {
		fmt.Println("No worker stores found in bucket.")
		return nil
	}

	for _, id := range workerIDs {
		fmt.Printf("Restoring worker %d store...\n", id)
		if err := restoreWorkerStore(ctx, client, id, reader); err != nil {
			return err
		}
	}
	return nil
}

// downloadStoreDir is a helper that downloads from <prefix>/<s3Dir>/ to a local directory
func downloadStoreDir(ctx context.Context, client *s3client.Client, s3Dir string, localDir string, label string, reader *bufio.Reader) error {
	// Check if local directory already has contents
	if info, err := os.Stat(localDir); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(localDir)
		if len(entries) > 0 {
			fmt.Printf("  %s already exists at %s with %d entries.\n", label, localDir, len(entries))
			if !promptYesNo(reader, fmt.Sprintf("  Overwrite existing %s contents?", label)) {
				fmt.Printf("  %s restore skipped.\n", label)
				return nil
			}
		}
	}

	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", label, err)
	}

	s3Prefix := client.GetObjectKey(s3Dir)

	keys, err := client.ListObjects(ctx, s3Prefix)
	if err != nil {
		return fmt.Errorf("failed to list %s objects: %w", label, err)
	}
	if len(keys) == 0 {
		fmt.Printf("  No %s files found in bucket, skipping.\n", label)
		return nil
	}

	fmt.Printf("  Downloading %s (%d files) -> %s\n", label, len(keys), localDir)

	count, err := client.DownloadDirectory(ctx, s3Prefix, localDir)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", label, err)
	}

	fmt.Printf("  Downloaded %d files.\n", count)
	return nil
}

// discoverWorkerIDs scans S3 keys under the worker-store prefix to find unique worker IDs.
// Keys are expected to look like "<prefix>/worker-store/1/somefile", "<prefix>/worker-store/2/somefile".
func discoverWorkerIDs(keys []string, wsPrefix string) []int {
	seen := make(map[int]bool)
	var ids []int

	for _, key := range keys {
		// Strip the worker-store prefix to get relative path like "1/somefile"
		rel := key
		if len(key) > len(wsPrefix) {
			rel = key[len(wsPrefix):]
		}

		// Extract the ID: "1/foo/bar" -> "1"
		slashIdx := strings.Index(rel, "/")
		var idStr string
		if slashIdx >= 0 {
			idStr = rel[:slashIdx]
		} else {
			idStr = rel
		}

		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			continue
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	return ids
}

// ============================================================================
// Path helpers
// ============================================================================

// getMasterStoreDir returns the local path to the master store.
// Local: $QUIL_NODE_PATH/.config/store/
func getMasterStoreDir() string {
	return filepath.Join(config.GetNodePath(), ".config", "store")
}

// getWorkerStoreBase returns the base directory for all worker stores.
// Local: $QUIL_NODE_PATH/.config/worker-store/
func getWorkerStoreBase() string {
	return filepath.Join(config.GetNodePath(), ".config", "worker-store")
}

// getWorkerStoreDir returns the local path to a specific worker's store.
// Local: $QUIL_NODE_PATH/.config/worker-store/<id>/
func getWorkerStoreDir(workerID int) string {
	return filepath.Join(getWorkerStoreBase(), strconv.Itoa(workerID))
}

// backupExistingConfig creates a backup of the existing config directory
func backupExistingConfig(configDir string) error {
	nodePath := config.GetNodePath()
	backupBase := filepath.Join(nodePath, ".config-bak")
	backupDir := backupBase

	if _, err := os.Stat(backupDir); err == nil {
		i := 1
		for {
			candidate := fmt.Sprintf("%s.%d", backupBase, i)
			if _, err := os.Stat(candidate); os.IsNotExist(err) {
				backupDir = candidate
				break
			}
			i++
		}
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	files := []string{"config.yml", "keys.yml"}
	for _, file := range files {
		src := filepath.Join(configDir, file)
		if _, err := os.Stat(src); err == nil {
			data, err := os.ReadFile(src)
			if err != nil {
				return fmt.Errorf("failed to read %s for backup: %w", file, err)
			}
			dst := filepath.Join(backupDir, file)
			if err := os.WriteFile(dst, data, 0644); err != nil {
				return fmt.Errorf("failed to write backup %s: %w", file, err)
			}
		}
	}

	fmt.Printf("Backed up existing config to %s\n", backupDir)
	return nil
}

// generateDefaultNodeConfig creates a minimal default node config file
func generateDefaultNodeConfig(path string) error {
	return createDefaultNodeConfigWithBinary(path, true)
}
