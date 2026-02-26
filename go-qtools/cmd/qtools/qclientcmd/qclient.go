package qclientcmd

import (
	"github.com/quilibrium/qtools/go-qtools/internal/config"
	"github.com/quilibrium/qtools/go-qtools/internal/node"
	"github.com/spf13/cobra"
)

// NewCommand builds the "qtools qclient" command tree.
func NewCommand() *cobra.Command {
	qclientCmd := &cobra.Command{
		Use:   "qclient",
		Short: "QClient management commands",
		Long:  "All commands related to QClient operations, including downloading binaries, token management (transfer, merge, split), and account operations.",
	}

	qclientDownloadCmd := &cobra.Command{
		Use:   "download [flags]",
		Short: "Download qclient binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			qtoolsConfigPath := config.GetConfigPath()
			qtoolsConfig, err := config.LoadConfig(qtoolsConfigPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			version, _ := cmd.Flags().GetString("version")
			return node.DownloadQClient(qtoolsConfig, version)
		},
	}
	qclientDownloadCmd.Flags().String("version", "", "Specific version to download (default: latest)")

	qclientCreateSymlinkCmd := &cobra.Command{
		Use:   "create-symlink",
		Short: "Create qclient symlink to downloaded binary",
		Long:  "Creates a symlink at /usr/local/bin/qclient pointing to the latest downloaded qclient binary.",
		RunE: func(cmd *cobra.Command, args []string) error {
			qtoolsConfigPath := config.GetConfigPath()
			qtoolsConfig, err := config.LoadConfig(qtoolsConfigPath)
			if err != nil {
				qtoolsConfig = config.GenerateDefaultConfig()
			}

			return node.CreateQClientSymlink(qtoolsConfig)
		},
	}

	qclientCmd.AddCommand(qclientDownloadCmd, qclientCreateSymlinkCmd)
	return qclientCmd
}
