package nodecmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/quilibrium/qtools/go-qtools/internal/config"
	"github.com/quilibrium/qtools/go-qtools/internal/node"
	"github.com/quilibrium/qtools/go-qtools/internal/tui"
	"github.com/spf13/cobra"
)

// NewCommand builds the "qtools node" command tree.
func NewCommand() *cobra.Command {
	nodeCmd := &cobra.Command{
		Use:   "node",
		Short: "Node management commands",
		Long:  "All commands related to Quilibrium node management, including setup, installation, updates, configuration, and status queries.",
	}

	setupCmd := &cobra.Command{
		Use:   "setup [flags]",
		Short: "Setup node (defaults to manual mode)",
		RunE: func(cmd *cobra.Command, args []string) error {
			automatic, _ := cmd.Flags().GetBool("automatic")
			workers, _ := cmd.Flags().GetInt("workers")

			fmt.Printf("Setting up node (automatic: %v, workers: %d)\n", automatic, workers)
			// TODO: Implement actual setup logic
			return nil
		},
	}
	setupCmd.Flags().Bool("automatic", false, "Use automatic mode instead of manual mode")
	setupCmd.Flags().Int("workers", 0, "Number of workers (0 = auto-calculate)")

	modeCmd := &cobra.Command{
		Use:   "mode [flags]",
		Short: "Toggle between manual and automatic mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			manual, _ := cmd.Flags().GetBool("manual")
			automatic, _ := cmd.Flags().GetBool("automatic")

			if manual && automatic {
				return fmt.Errorf("cannot specify both --manual and --automatic")
			}

			fmt.Printf("Toggling mode (manual: %v, automatic: %v)\n", manual, automatic)
			// TODO: Implement actual mode toggle logic
			return nil
		},
	}
	modeCmd.Flags().Bool("manual", false, "Enable manual mode")
	modeCmd.Flags().Bool("automatic", false, "Enable automatic mode")

	installCmd := &cobra.Command{
		Use:   "install [flags]",
		Short: "Complete installation of the node",
		RunE: func(cmd *cobra.Command, args []string) error {
			peerID, _ := cmd.Flags().GetString("peer-id")
			listenPort, _ := cmd.Flags().GetInt("listen-port")
			streamPort, _ := cmd.Flags().GetInt("stream-port")
			baseP2PPort, _ := cmd.Flags().GetInt("base-p2p-port")
			baseStreamPort, _ := cmd.Flags().GetInt("base-stream-port")

			fmt.Printf("Installing node (peer-id: %s, listen-port: %d, stream-port: %d, base-p2p-port: %d, base-stream-port: %d)\n",
				peerID, listenPort, streamPort, baseP2PPort, baseStreamPort)

			qtoolsConfigPath := config.GetConfigPath()
			qtoolsConfig, err := config.LoadConfig(qtoolsConfigPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			opts := node.InstallOptions{
				PeerID:         peerID,
				ListenPort:     listenPort,
				StreamPort:     streamPort,
				BaseP2PPort:    baseP2PPort,
				BaseStreamPort: baseStreamPort,
			}

			if err := node.CompleteInstall(opts, qtoolsConfig); err != nil {
				return err
			}

			fmt.Println("Node installation completed successfully")
			return nil
		},
	}
	installCmd.Flags().String("peer-id", "", "Peer ID for the node")
	installCmd.Flags().Int("listen-port", 8336, "P2P listen port")
	installCmd.Flags().Int("stream-port", 8340, "Stream listen port")
	installCmd.Flags().Int("base-p2p-port", 25000, "Base P2P port for manual workers")
	installCmd.Flags().Int("base-stream-port", 32500, "Base stream port for manual workers")

	nodeConfigCmd := &cobra.Command{
		Use:   "config [path]",
		Short: "Browse node configuration (TUI)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			qtoolsConfigPath := config.GetConfigPath()
			qtoolsConfig, err := config.LoadConfig(qtoolsConfigPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			initialPath := ""
			if len(args) > 0 {
				initialPath = args[0]
			}

			return tui.RunConfigView(qtoolsConfig, initialPath, tui.ConfigTypeQuil)
		},
	}

	nodeConfigGetCmd := &cobra.Command{
		Use:   "get <path>",
		Short: "Get node config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			qtoolsConfigPath := config.GetConfigPath()
			qtoolsConfig, err := config.LoadConfig(qtoolsConfigPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			configType, _ := cmd.Flags().GetString("config")
			defaultVal, _ := cmd.Flags().GetString("default")
			opts := node.ConfigCommandOptions{ConfigType: configType, Default: defaultVal}

			return node.ExecuteConfigCommand(node.ConfigCommandGet, args[0], "", opts, qtoolsConfig)
		},
	}
	nodeConfigGetCmd.Flags().String("config", "quil", "Config type: qtools or quil (default: quil for node config)")
	nodeConfigGetCmd.Flags().String("default", "", "Default value if key not found")

	nodeConfigSetCmd := &cobra.Command{
		Use:   "set <path> <value>",
		Short: "Set node config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			qtoolsConfigPath := config.GetConfigPath()
			qtoolsConfig, err := config.LoadConfig(qtoolsConfigPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			configType, _ := cmd.Flags().GetString("config")
			quiet, _ := cmd.Flags().GetBool("quiet")
			opts := node.ConfigCommandOptions{ConfigType: configType, Quiet: quiet}

			return node.ExecuteConfigCommand(node.ConfigCommandSet, args[0], args[1], opts, qtoolsConfig)
		},
	}
	nodeConfigSetCmd.Flags().String("config", "qtools", "Config type: qtools or quil")
	nodeConfigSetCmd.Flags().Bool("quiet", false, "Suppress output")

	nodeConfigCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create default node config",
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			nodeConfigPath, err := node.CreateDefaultNodeConfig(force)
			if err != nil {
				return err
			}
			fmt.Printf("Default node config created at %s\n", nodeConfigPath)
			return nil
		},
	}
	nodeConfigCreateCmd.Flags().Bool("force", false, "Overwrite existing node config")
	nodeConfigCmd.AddCommand(nodeConfigGetCmd, nodeConfigSetCmd, nodeConfigCreateCmd)

	nodeInfoCmd := &cobra.Command{
		Use:   "info",
		Short: "Get node information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Node information:")
			return nil
		},
	}

	nodePeerIDCmd := &cobra.Command{
		Use:   "peer-id",
		Short: "Get node peer ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Peer ID:")
			return nil
		},
	}

	nodeBalanceCmd := &cobra.Command{
		Use:   "balance",
		Short: "Get node balance",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Balance:")
			return nil
		},
	}

	nodeSeniorityCmd := &cobra.Command{
		Use:   "seniority",
		Short: "Get node seniority",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Seniority:")
			return nil
		},
	}

	nodeWorkerCountCmd := &cobra.Command{
		Use:   "worker-count",
		Short: "Get worker count",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Worker count:")
			return nil
		},
	}

	nodeUpdateCmd := &cobra.Command{
		Use:   "update [flags]",
		Short: "Update node binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			qtoolsConfigPath := config.GetConfigPath()
			qtoolsConfig, err := config.LoadConfig(qtoolsConfigPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			force, _ := cmd.Flags().GetBool("force")
			skipClean, _ := cmd.Flags().GetBool("skip-clean")
			auto, _ := cmd.Flags().GetBool("auto")
			opts := node.UpdateOptions{Force: force, SkipClean: skipClean, Auto: auto}

			if err := node.UpdateNode(opts, qtoolsConfig); err != nil {
				return err
			}

			fmt.Println("Node update completed successfully")
			return nil
		},
	}
	nodeUpdateCmd.Flags().Bool("force", false, "Force update")
	nodeUpdateCmd.Flags().Bool("skip-clean", false, "Skip cleanup")
	nodeUpdateCmd.Flags().Bool("auto", false, "Auto-update mode")

	nodeDownloadCmd := &cobra.Command{
		Use:   "download [flags]",
		Short: "Download node binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			qtoolsConfigPath := config.GetConfigPath()
			qtoolsConfig, err := config.LoadConfig(qtoolsConfigPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			version, _ := cmd.Flags().GetString("version")
			link, _ := cmd.Flags().GetBool("link")
			return node.DownloadNode(qtoolsConfig, version, link)
		},
	}
	nodeDownloadCmd.Flags().String("version", "", "Specific version to download (default: latest)")
	nodeDownloadCmd.Flags().Bool("link", false, "Create symlink after download")

	nodeBackupCmd := &cobra.Command{
		Use:   "backup [flags]",
		Short: "Backup node data to QStorage (S3-compatible bucket)",
		RunE: func(cmd *cobra.Command, args []string) error {
			qtoolsConfigPath := config.GetConfigPath()
			qtoolsConfig, err := config.LoadConfig(qtoolsConfigPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			opts, err := parseBackupRestoreFlags(cmd)
			if err != nil {
				return err
			}

			if err := node.BackupNode(opts, qtoolsConfig); err != nil {
				return err
			}

			reader := bufio.NewReader(os.Stdin)
			if err := node.PromptSaveCredentials(opts, qtoolsConfig, reader); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}

			return nil
		},
	}
	addBackupRestoreFlags(nodeBackupCmd)

	nodeRestoreCmd := &cobra.Command{
		Use:   "restore [flags]",
		Short: "Restore node data from QStorage (S3-compatible bucket)",
		RunE: func(cmd *cobra.Command, args []string) error {
			qtoolsConfigPath := config.GetConfigPath()
			qtoolsConfig, err := config.LoadConfig(qtoolsConfigPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			opts, err := parseBackupRestoreFlags(cmd)
			if err != nil {
				return err
			}

			if err := node.RestoreNode(opts, qtoolsConfig); err != nil {
				return err
			}

			reader := bufio.NewReader(os.Stdin)
			if err := node.PromptSaveCredentials(opts, qtoolsConfig, reader); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}

			return nil
		},
	}
	addBackupRestoreFlags(nodeRestoreCmd)
	nodeRestoreCmd.Flags().Bool("keys-only", false, "Only migrate key data (modifier for --config-files)")

	nodeCmd.AddCommand(
		setupCmd,
		modeCmd,
		installCmd,
		nodeConfigCmd,
		nodeInfoCmd,
		nodePeerIDCmd,
		nodeBalanceCmd,
		nodeSeniorityCmd,
		nodeWorkerCountCmd,
		nodeUpdateCmd,
		nodeDownloadCmd,
		nodeBackupCmd,
		nodeRestoreCmd,
	)

	return nodeCmd
}

func addBackupRestoreFlags(cmd *cobra.Command) {
	cmd.Flags().String("access-key-id", "", "QStorage/S3 access key ID")
	cmd.Flags().String("access-key", "", "QStorage/S3 access key")
	cmd.Flags().String("account-id", "", "QStorage/S3 account ID")
	cmd.Flags().String("bucket", "", "S3 bucket name")
	cmd.Flags().String("region", "", "S3 region (default: q-world-1)")
	cmd.Flags().String("endpoint-url", "", "S3 endpoint URL (default: https://qstorage.quilibrium.com)")
	cmd.Flags().String("prefix", "", "S3 key prefix/path for bucket root")
	cmd.Flags().Bool("config-files", false, "Only operate on config files (config.yml + keys.yml)")
	cmd.Flags().Bool("master", false, "Only operate on the master store")
	cmd.Flags().String("worker", "", "Only operate on specific worker store(s) (comma-separated IDs, e.g. 1,2,3)")
}

func parseBackupRestoreFlags(cmd *cobra.Command) (node.BackupRestoreOptions, error) {
	accessKeyID, _ := cmd.Flags().GetString("access-key-id")
	accessKey, _ := cmd.Flags().GetString("access-key")
	accountID, _ := cmd.Flags().GetString("account-id")
	bucket, _ := cmd.Flags().GetString("bucket")
	region, _ := cmd.Flags().GetString("region")
	endpointURL, _ := cmd.Flags().GetString("endpoint-url")
	prefix, _ := cmd.Flags().GetString("prefix")
	configFiles, _ := cmd.Flags().GetBool("config-files")
	keysOnly, _ := cmd.Flags().GetBool("keys-only")
	master, _ := cmd.Flags().GetBool("master")
	workerCSV, _ := cmd.Flags().GetString("worker")

	workers, err := node.ParseWorkerIDs(workerCSV)
	if err != nil {
		return node.BackupRestoreOptions{}, fmt.Errorf("invalid --worker value: %w", err)
	}

	return node.BackupRestoreOptions{
		AccessKeyID: accessKeyID,
		AccessKey:   accessKey,
		AccountID:   accountID,
		Bucket:      bucket,
		Region:      region,
		EndpointURL: endpointURL,
		Prefix:      prefix,
		ConfigFiles: configFiles,
		KeysOnly:    keysOnly,
		Master:      master,
		Workers:     workers,
	}, nil
}
