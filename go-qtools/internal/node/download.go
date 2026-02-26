package node

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/quilibrium/qtools/go-qtools/internal/config"
)

// DownloadNode downloads the node binary (optionally with version and link)
func DownloadNode(cfg *config.Config, version string, createLink bool) error {
	if version == "" {
		// Get latest version
		var err error
		version, err = FetchNodeReleaseVersion()
		if err != nil {
			return fmt.Errorf("failed to fetch node version: %w", err)
		}
	}

	osArch := getOSArch()
	nodePath := config.GetNodePath()

	// Ensure directory exists
	if err := os.MkdirAll(nodePath, 0755); err != nil {
		return fmt.Errorf("failed to create node directory: %w", err)
	}

	binaryName := fmt.Sprintf("node-%s-%s", version, osArch)
	binaryPath := filepath.Join(nodePath, binaryName)

	// Check if binary already exists
	if _, err := os.Stat(binaryPath); err == nil {
		fmt.Printf("Node binary %s already exists\n", binaryName)
	} else {
		// Download binary
		url := fmt.Sprintf("https://releases.quilibrium.com/%s", binaryName)
		fmt.Printf("Downloading %s...\n", url)

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to download binary: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download binary: status %d", resp.StatusCode)
		}

		// Create file
		out, err := os.Create(binaryPath)
		if err != nil {
			return fmt.Errorf("failed to create binary file: %w", err)
		}
		defer out.Close()

		// Copy to file
		if _, err := io.Copy(out, resp.Body); err != nil {
			return fmt.Errorf("failed to write binary: %w", err)
		}

		// Make executable
		if err := os.Chmod(binaryPath, 0755); err != nil {
			return fmt.Errorf("failed to make binary executable: %w", err)
		}
	}

	// Set ownership
	if err := setFileOwnership(binaryPath); err != nil {
		fmt.Printf("Warning: failed to set ownership: %v\n", err)
	}

	fmt.Printf("✓ Node binary downloaded: %s\n", binaryName)

	// Create symlink if requested
	if createLink {
		if err := updateNodeSymlink(binaryPath); err != nil {
			return err
		}

		fmt.Printf("✓ Created symlink: %s -> %s\n", "/usr/local/bin/quilibrium-node", binaryPath)

		// Update version in config
		if cfg != nil {
			cfg.CurrentNodeVersion = version
			configPath := config.GetConfigPath()
			if err := config.SaveConfig(cfg, configPath); err != nil {
				fmt.Printf("Warning: failed to save version to config: %v\n", err)
			}
		}
	}

	return nil
}

// DownloadQClient downloads the qclient binary
func DownloadQClient(cfg *config.Config, version string) error {
	if version == "" {
		// Get latest version
		var err error
		version, err = fetchQClientReleaseVersion()
		if err != nil {
			return fmt.Errorf("failed to fetch qclient version: %w", err)
		}
	}

	osArch := getOSArch()
	clientPath := config.GetClientPath()

	// Ensure directory exists
	if err := os.MkdirAll(clientPath, 0755); err != nil {
		return fmt.Errorf("failed to create client directory: %w", err)
	}

	binaryName := fmt.Sprintf("qclient-%s-%s", version, osArch)
	binaryPath := filepath.Join(clientPath, binaryName)

	// Check if binary already exists
	if _, err := os.Stat(binaryPath); err == nil {
		fmt.Printf("QClient binary %s already exists\n", binaryName)
	} else {
		// Download binary
		url := fmt.Sprintf("https://releases.quilibrium.com/%s", binaryName)
		fmt.Printf("Downloading %s...\n", url)

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to download binary: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download binary: status %d", resp.StatusCode)
		}

		// Create file
		out, err := os.Create(binaryPath)
		if err != nil {
			return fmt.Errorf("failed to create binary file: %w", err)
		}
		defer out.Close()

		// Copy to file
		if _, err := io.Copy(out, resp.Body); err != nil {
			return fmt.Errorf("failed to write binary: %w", err)
		}

		// Make executable
		if err := os.Chmod(binaryPath, 0755); err != nil {
			return fmt.Errorf("failed to make binary executable: %w", err)
		}
	}

	// Set ownership
	if err := setFileOwnership(binaryPath); err != nil {
		fmt.Printf("Warning: failed to set ownership: %v\n", err)
	}

	fmt.Printf("✓ QClient binary downloaded: %s\n", binaryName)
	return nil
}
