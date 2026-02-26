package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/quilibrium/qtools/go-qtools/cmd/qtools/nodecmd"
	"github.com/quilibrium/qtools/go-qtools/cmd/qtools/qclientcmd"
	"github.com/quilibrium/qtools/go-qtools/internal/config"
	"github.com/quilibrium/qtools/go-qtools/internal/node"
	"github.com/quilibrium/qtools/go-qtools/internal/tui"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
)

func main() {
	// Detect if binary was called as "qclient" (via symlink or alias)
	binaryName := filepath.Base(os.Args[0])
	if binaryName == "qclient" {
		// Prepend "qclient" to args to route to qclient subcommand
		os.Args = append([]string{os.Args[0], "qclient"}, os.Args[1:]...)
	}

	rootCmd := &cobra.Command{
		Use:     "qtools",
		Short:   "Quilibrium Tools - Node management CLI",
		Long:    "Quilibrium Tools provides CLI and TUI interfaces for managing Quilibrium nodes.",
		Version: version,
	}

	// Node commands (extracted into dedicated package)
	nodeCmd := nodecmd.NewCommand()

	// Service commands
	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "Service management commands",
	}

	startCmd := &cobra.Command{
		Use:   "start [flags]",
		Short: "Start service(s)",
		RunE: func(cmd *cobra.Command, args []string) error {
			master, _ := cmd.Flags().GetBool("master")
			coreIndex, _ := cmd.Flags().GetInt("core-index")
			cores, _ := cmd.Flags().GetString("cores")

			fmt.Printf("Starting service (master: %v, core-index: %d, cores: %s)\n", master, coreIndex, cores)
			// TODO: Implement actual start logic
			return nil
		},
	}
	startCmd.Flags().Bool("master", false, "Start master only")
	startCmd.Flags().Int("core-index", 0, "Start specific worker by core index")
	startCmd.Flags().String("cores", "", "Start specific workers by core numbers (e.g., '1-4,6,8')")

	stopCmd := &cobra.Command{
		Use:   "stop [flags]",
		Short: "Stop service(s)",
		RunE: func(cmd *cobra.Command, args []string) error {
			master, _ := cmd.Flags().GetBool("master")
			coreIndex, _ := cmd.Flags().GetInt("core-index")
			cores, _ := cmd.Flags().GetString("cores")

			fmt.Printf("Stopping service (master: %v, core-index: %d, cores: %s)\n", master, coreIndex, cores)
			// TODO: Implement actual stop logic
			return nil
		},
	}
	stopCmd.Flags().Bool("master", false, "Stop master only")
	stopCmd.Flags().Int("core-index", 0, "Stop specific worker by core index")
	stopCmd.Flags().String("cores", "", "Stop specific workers by core numbers")

	restartCmd := &cobra.Command{
		Use:   "restart [flags]",
		Short: "Restart service(s)",
		RunE: func(cmd *cobra.Command, args []string) error {
			master, _ := cmd.Flags().GetBool("master")
			coreIndex, _ := cmd.Flags().GetInt("core-index")
			cores, _ := cmd.Flags().GetString("cores")

			fmt.Printf("Restarting service (master: %v, core-index: %d, cores: %s)\n", master, coreIndex, cores)
			// TODO: Implement actual restart logic
			return nil
		},
	}
	restartCmd.Flags().Bool("master", false, "Restart master only")
	restartCmd.Flags().Int("core-index", 0, "Restart specific worker by core index")
	restartCmd.Flags().String("cores", "", "Restart specific workers by core numbers")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Get service status",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Service status:")
			// TODO: Implement actual status logic
			return nil
		},
	}

	serviceEnableCmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable service on boot",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Enabling service...")
			// TODO: Implement service enable
			return nil
		},
	}

	serviceDisableCmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable service on boot",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Disabling service...")
			// TODO: Implement service disable
			return nil
		},
	}

	serviceUpdateCmd := &cobra.Command{
		Use:   "update [flags]",
		Short: "Update service configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Updating service configuration...")
			// TODO: Implement service update
			return nil
		},
	}

	serviceCmd.AddCommand(startCmd, stopCmd, restartCmd, statusCmd, serviceEnableCmd, serviceDisableCmd, serviceUpdateCmd)

	// Backup commands
	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup and restore commands",
	}

	backupPeerCmd := &cobra.Command{
		Use:   "peer [flags]",
		Short: "Backup peer config files",
		RunE: func(cmd *cobra.Command, args []string) error {
			peerID, _ := cmd.Flags().GetString("peer-id")
			local, _ := cmd.Flags().GetString("local")
			fmt.Printf("Backing up peer config (peer-id: %s, local: %s)\n", peerID, local)
			// TODO: Implement backup peer
			return nil
		},
	}
	backupPeerCmd.Flags().String("peer-id", "", "Peer ID for backup")
	backupPeerCmd.Flags().String("local", "", "Backup to local directory")

	backupStoreCmd := &cobra.Command{
		Use:   "store [flags]",
		Short: "Backup store directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Backing up store directory...")
			// TODO: Implement backup store
			return nil
		},
	}

	backupLocalCmd := &cobra.Command{
		Use:   "local [flags]",
		Short: "Create local backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Creating local backup...")
			// TODO: Implement local backup
			return nil
		},
	}

	backupRestoreCmd := &cobra.Command{
		Use:   "restore [flags]",
		Short: "Restore complete backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			peerID, _ := cmd.Flags().GetString("peer-id")
			noStore, _ := cmd.Flags().GetBool("no-store")
			fmt.Printf("Restoring backup (peer-id: %s, no-store: %v)\n", peerID, noStore)
			// TODO: Implement restore
			return nil
		},
	}
	backupRestoreCmd.Flags().String("peer-id", "", "Peer ID to restore")
	backupRestoreCmd.Flags().Bool("no-store", false, "Skip store restore")

	backupCmd.AddCommand(backupPeerCmd, backupStoreCmd, backupLocalCmd, backupRestoreCmd)

	// Diagnostics commands
	diagnosticsCmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "Diagnostic commands",
	}

	diagnosticsStatusReportCmd := &cobra.Command{
		Use:   "status-report [flags]",
		Short: "Generate comprehensive status report",
		RunE: func(cmd *cobra.Command, args []string) error {
			json, _ := cmd.Flags().GetBool("json")
			fmt.Printf("Generating status report (json: %v)\n", json)
			// TODO: Implement status report
			return nil
		},
	}
	diagnosticsStatusReportCmd.Flags().Bool("json", false, "Output in JSON format")

	diagnosticsCheckFilesCmd := &cobra.Command{
		Use:   "check-files",
		Short: "Check node file integrity",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Checking node files...")
			// TODO: Implement check files
			return nil
		},
	}

	diagnosticsCheckPortsCmd := &cobra.Command{
		Use:   "check-ports",
		Short: "Check listening ports",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Checking ports...")
			// TODO: Implement check ports
			return nil
		},
	}

	diagnosticsRunCmd := &cobra.Command{
		Use:   "run",
		Short: "Run all diagnostics",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Running all diagnostics...")
			// TODO: Implement run diagnostics
			return nil
		},
	}

	diagnosticsCmd.AddCommand(diagnosticsStatusReportCmd, diagnosticsCheckFilesCmd, diagnosticsCheckPortsCmd, diagnosticsRunCmd)

	// Update commands
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update commands",
	}

	updateSelfCmd := &cobra.Command{
		Use:   "self [flags]",
		Short: "Update qtools itself (check for updates)",
		Long:  "Checks for qtools updates and optionally updates the binary. Use --check to only check without updating.",
		RunE: func(cmd *cobra.Command, args []string) error {
			checkOnly, _ := cmd.Flags().GetBool("check")
			_, _ = cmd.Flags().GetBool("auto") // Reserved for future use

			if checkOnly {
				fmt.Println("Checking for qtools updates...")
				// TODO: Implement update check
				fmt.Println("Update check not yet implemented")
				return nil
			}

			fmt.Println("Updating qtools...")
			fmt.Println("Self-update not yet implemented - qtools is now a binary")
			fmt.Println("To update qtools, download the latest binary from the releases page")
			// TODO: Implement self-update for binary
			return nil
		},
	}
	updateSelfCmd.Flags().Bool("check", false, "Only check for updates, don't update")
	updateSelfCmd.Flags().Bool("auto", false, "Run in auto mode")

	updateKernelCmd := &cobra.Command{
		Use:   "kernel",
		Short: "Update Linux kernel",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Updating kernel...")
			// TODO: Implement kernel update
			return nil
		},
	}

	updateCmd.AddCommand(updateSelfCmd, updateKernelCmd)

	// QClient commands (extracted into dedicated package)
	qclientCmd := qclientcmd.NewCommand()

	// Toggle commands
	toggleCmd := &cobra.Command{
		Use:   "toggle",
		Short: "Toggle configuration settings",
	}

	toggleAutoUpdateNodeCmd := &cobra.Command{
		Use:   "auto-update-node [flags]",
		Short: "Toggle node auto-updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			configPath := os.Getenv("QTOOLS_CONFIG_FILE")
			if configPath == "" {
				configPath = "/home/quilibrium/qtools/config.yml"
			}

			qtoolsConfig, err := config.LoadConfig(configPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			on, _ := cmd.Flags().GetBool("on")
			off, _ := cmd.Flags().GetBool("off")

			// Get current status using config path
			path := "scheduled_tasks.updates.node.enabled"
			currentVal, err := config.GetConfigValue(qtoolsConfig, path)
			currentStatus := false
			if err == nil {
				if val, ok := currentVal.(bool); ok {
					currentStatus = val
				}
			}

			var newStatus bool
			if on && off {
				return fmt.Errorf("cannot specify both --on and --off")
			} else if on {
				if currentStatus {
					fmt.Println("Node auto-updates are already enabled.")
					return nil
				}
				newStatus = true
			} else if off {
				if !currentStatus {
					fmt.Println("Node auto-updates are already disabled.")
					return nil
				}
				newStatus = false
			} else {
				// Toggle
				newStatus = !currentStatus
			}

			// Set new status
			if err := config.SetConfigValue(qtoolsConfig, path, newStatus); err != nil {
				return fmt.Errorf("failed to set config value: %w", err)
			}

			// Save config
			if err := config.SaveConfig(qtoolsConfig, configPath); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			statusText := "off"
			if newStatus {
				statusText = "on"
			}
			fmt.Printf("Node auto-updates have been turned %s.\n", statusText)

			// TODO: Update cron (when install-cron is implemented)
			// fmt.Println("Updating cron tasks...")

			return nil
		},
	}
	toggleAutoUpdateNodeCmd.Flags().Bool("on", false, "Enable auto-updates")
	toggleAutoUpdateNodeCmd.Flags().Bool("off", false, "Disable auto-updates")

	toggleAutoUpdateQtoolsCmd := &cobra.Command{
		Use:   "auto-update-qtools [flags]",
		Short: "Toggle qtools auto-updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			configPath := os.Getenv("QTOOLS_CONFIG_FILE")
			if configPath == "" {
				configPath = "/home/quilibrium/qtools/config.yml"
			}

			qtoolsConfig, err := config.LoadConfig(configPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			on, _ := cmd.Flags().GetBool("on")
			off, _ := cmd.Flags().GetBool("off")

			// Get current status using config path
			path := "scheduled_tasks.updates.qtools.enabled"
			currentVal, err := config.GetConfigValue(qtoolsConfig, path)
			currentStatus := false
			if err == nil {
				if val, ok := currentVal.(bool); ok {
					currentStatus = val
				}
			}

			var newStatus bool
			if on && off {
				return fmt.Errorf("cannot specify both --on and --off")
			} else if on {
				if currentStatus {
					fmt.Println("Qtools auto-updates are already enabled.")
					return nil
				}
				newStatus = true
			} else if off {
				if !currentStatus {
					fmt.Println("Qtools auto-updates are already disabled.")
					return nil
				}
				newStatus = false
			} else {
				// Toggle
				newStatus = !currentStatus
			}

			// Set new status
			if err := config.SetConfigValue(qtoolsConfig, path, newStatus); err != nil {
				return fmt.Errorf("failed to set config value: %w", err)
			}

			// Save config
			if err := config.SaveConfig(qtoolsConfig, configPath); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			statusText := "off"
			if newStatus {
				statusText = "on"
			}
			fmt.Printf("Qtools auto-updates have been turned %s.\n", statusText)

			// TODO: Update cron (when install-cron is implemented)

			return nil
		},
	}
	toggleAutoUpdateQtoolsCmd.Flags().Bool("on", false, "Enable auto-updates")
	toggleAutoUpdateQtoolsCmd.Flags().Bool("off", false, "Disable auto-updates")

	toggleCmd.AddCommand(toggleAutoUpdateNodeCmd, toggleAutoUpdateQtoolsCmd)

	// Utility commands
	utilCmd := &cobra.Command{
		Use:   "util",
		Short: "Utility commands",
	}

	publicIPCmd := &cobra.Command{
		Use:   "public-ip",
		Short: "Get public IP address",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Try multiple services to get public IP
			services := []string{
				"https://api.ipify.org",
				"https://icanhazip.com",
				"https://ifconfig.me",
			}

			for _, service := range services {
				resp, err := http.Get(service)
				if err != nil {
					continue
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					body, err := io.ReadAll(resp.Body)
					if err == nil {
						ip := strings.TrimSpace(string(body))
						fmt.Println(ip)
						return nil
					}
				}
			}

			return fmt.Errorf("failed to get public IP from any service")
		},
	}

	utilCmd.AddCommand(publicIPCmd)

	// Log commands
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Log viewing and management",
	}

	logsViewCmd := &cobra.Command{
		Use:   "view [flags]",
		Short: "View logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			master, _ := cmd.Flags().GetBool("master")
			worker, _ := cmd.Flags().GetInt("worker")
			qtools, _ := cmd.Flags().GetBool("qtools")
			fmt.Printf("Viewing logs (master: %v, worker: %d, qtools: %v)\n", master, worker, qtools)
			// TODO: Implement log view
			return nil
		},
	}
	logsViewCmd.Flags().Bool("master", false, "View master log")
	logsViewCmd.Flags().Int("worker", 0, "View worker log (specify worker number)")
	logsViewCmd.Flags().Bool("qtools", false, "View qtools log")

	logsConfigureCmd := &cobra.Command{
		Use:   "configure [flags]",
		Short: "Configure custom logging",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Configuring logging...")
			// TODO: Implement log configure
			return nil
		},
	}

	logsCmd.AddCommand(logsViewCmd, logsConfigureCmd)

	// Config commands (for qtools config, not node config)
	// Top-level config command - Always for qtools config
	configCmd := &cobra.Command{
		Use:   "config [path]",
		Short: "Browse qtools configuration (TUI)",
		Long: `Browse qtools configuration files in TUI mode.

If no path is provided, shows top-level keys.
If a path is provided (e.g., "service"), navigates directly to that section.

Use subcommands for CLI operations (bypasses TUI):
  qtools config get <path>   # Get config value (CLI)
  qtools config set <path> <value>  # Set config value (CLI)

Examples:
  qtools config              # Browse qtools config from root (TUI)
  qtools config service       # Navigate to service section (TUI)
  qtools config get scheduled_tasks.updates.node.enabled  # Get value via CLI (no TUI)
  qtools config set scheduled_tasks.updates.node.enabled true  # Set value via CLI (no TUI)
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			configPath := os.Getenv("QTOOLS_CONFIG_FILE")
			if configPath == "" {
				configPath = "/home/quilibrium/qtools/config.yml"
			}

			qtoolsConfig, err := config.LoadConfig(configPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			initialPath := ""
			if len(args) > 0 {
				initialPath = args[0]
			}

			// Always use qtools config for top-level config command
			return tui.RunConfigView(qtoolsConfig, initialPath, tui.ConfigTypeQtools)
		},
	}

	configGetCmd := &cobra.Command{
		Use:   "get <path>",
		Short: "Get qtools config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			configPath := os.Getenv("QTOOLS_CONFIG_FILE")
			if configPath == "" {
				configPath = "/home/quilibrium/qtools/config.yml"
			}

			qtoolsConfig, err := config.LoadConfig(configPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			defaultVal, _ := cmd.Flags().GetString("default")

			opts := node.ConfigCommandOptions{
				ConfigType: "qtools",
				Default:    defaultVal,
			}

			return node.ExecuteConfigCommand(node.ConfigCommandGet, args[0], "", opts, qtoolsConfig)
		},
	}
	configGetCmd.Flags().String("default", "", "Default value if key not found")

	configSetCmd := &cobra.Command{
		Use:   "set <path> <value>",
		Short: "Set qtools config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			configPath := os.Getenv("QTOOLS_CONFIG_FILE")
			if configPath == "" {
				configPath = "/home/quilibrium/qtools/config.yml"
			}

			qtoolsConfig, err := config.LoadConfig(configPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			quiet, _ := cmd.Flags().GetBool("quiet")

			opts := node.ConfigCommandOptions{
				ConfigType: "qtools",
				Quiet:      quiet,
			}

			return node.ExecuteConfigCommand(node.ConfigCommandSet, args[0], args[1], opts, qtoolsConfig)
		},
	}
	configSetCmd.Flags().Bool("quiet", false, "Suppress output")

	configCmd.AddCommand(configGetCmd, configSetCmd)

	// Completion command
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Install shell completion script",
		Long: `Install shell completion script for qtools permanently.

If no shell is specified, qtools will auto-detect your shell and install
completions automatically. If detection fails, you'll be prompted to select
your shell.

To generate completion script without installing (for manual installation):
  qtools completion bash --generate

Supported shells: bash, zsh, fish, powershell
`,
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			generateOnly, _ := cmd.Flags().GetBool("generate")

			var shell string
			if len(args) > 0 {
				shell = args[0]
			} else {
				// Auto-detect shell
				shell = detectShell()
				if shell == "" {
					// Prompt user
					var err error
					shell, err = promptShell()
					if err != nil {
						return fmt.Errorf("failed to detect shell: %w", err)
					}
				}
			}

			// Validate shell
			validShells := map[string]bool{"bash": true, "zsh": true, "fish": true, "powershell": true}
			if !validShells[shell] {
				return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish, powershell)", shell)
			}

			// If --generate flag, just output to stdout
			if generateOnly {
				switch shell {
				case "bash":
					return rootCmd.GenBashCompletion(os.Stdout)
				case "zsh":
					return rootCmd.GenZshCompletion(os.Stdout)
				case "fish":
					return rootCmd.GenFishCompletion(os.Stdout, true)
				case "powershell":
					return rootCmd.GenPowerShellCompletion(os.Stdout)
				}
			}

			// Otherwise, install permanently
			if shell == "powershell" {
				// PowerShell is handled differently - output instructions
				fmt.Println("PowerShell completion:")
				fmt.Println("  qtools completion powershell --generate | Out-String | Invoke-Expression")
				fmt.Println("\nOr add to your PowerShell profile:")
				fmt.Println("  qtools completion powershell --generate | Out-String | Add-Content $PROFILE")
				return rootCmd.GenPowerShellCompletion(os.Stdout)
			}

			return installCompletion(rootCmd, shell)
		},
	}
	completionCmd.Flags().Bool("generate", false, "Generate completion script to stdout instead of installing")

	// TUI command
	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch TUI mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			configPath := os.Getenv("QTOOLS_CONFIG_FILE")
			if configPath == "" {
				configPath = "/home/quilibrium/qtools/config.yml"
			}

			qtoolsConfig, err := config.LoadConfig(configPath)
			if err != nil {
				// Create default config if it doesn't exist
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			// Run TUI
			return tui.Run(qtoolsConfig)
		},
	}

	rootCmd.AddCommand(nodeCmd, serviceCmd, backupCmd, diagnosticsCmd, updateCmd, logsCmd, configCmd, qclientCmd, toggleCmd, utilCmd, completionCmd, tuiCmd)

	// Register custom completions
	registerCompletions(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
