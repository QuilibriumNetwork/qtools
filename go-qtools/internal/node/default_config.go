package node

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/quilibrium/qtools/go-qtools/internal/config"
)

// CreateDefaultNodeConfig creates a default node config at the expected path.
func CreateDefaultNodeConfig(overwrite bool) (string, error) {
	configPath := config.GetNodeConfigPath()
	if err := createDefaultNodeConfigWithBinary(configPath, overwrite); err != nil {
		return "", err
	}
	return configPath, nil
}

func createDefaultNodeConfigWithBinary(configPath string, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("node config already exists at %s (use --force to overwrite)", configPath)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat config path: %w", err)
		}
	} else if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing node config: %w", err)
	}

	if err := os.MkdirAll(config.GetNodePath(), 0755); err != nil {
		return fmt.Errorf("failed to create node directory: %w", err)
	}

	if err := ensureQuilibriumNodeSymlink(); err != nil {
		return err
	}

	cmd := exec.Command("/usr/local/bin/quilibrium-node", "--node-info")
	cmd.Dir = config.GetNodePath()
	cmd.Env = append(os.Environ(), "QUIL_CONFIG_FILE="+configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Some node binaries may return non-zero while still generating config.
		if _, statErr := os.Stat(configPath); statErr == nil {
			return nil
		}
		msg := strings.TrimSpace(string(output))
		if msg != "" {
			return fmt.Errorf("failed to generate node config via quilibrium-node --node-info: %w\nOutput: %s", err, msg)
		}
		return fmt.Errorf("failed to generate node config via quilibrium-node --node-info: %w", err)
	}

	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("node config was not generated at %s", configPath)
	}

	return nil
}

func ensureQuilibriumNodeSymlink() error {
	const symlinkPath = "/usr/local/bin/quilibrium-node"

	if target, err := os.Readlink(symlinkPath); err == nil {
		if _, statErr := os.Stat(target); statErr == nil {
			return nil
		}
	}

	nodeBinary, err := findLatestNodeBinary(config.GetNodePath(), getOSArch())
	if err != nil {
		return fmt.Errorf("quilibrium-node symlink is missing and no downloaded node binary was found in %s; run `qtools node update` first", config.GetNodePath())
	}

	cmd := exec.Command("sudo", "ln", "-sf", nodeBinary, symlinkPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(output))
		if msg != "" {
			return fmt.Errorf("failed to create quilibrium-node symlink: %w\nOutput: %s", err, msg)
		}
		return fmt.Errorf("failed to create quilibrium-node symlink: %w", err)
	}

	return nil
}
