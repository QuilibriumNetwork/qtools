package node

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// updateNodeSymlink force-updates /usr/local/bin/quilibrium-node to binaryPath.
func updateNodeSymlink(binaryPath string) error {
	const symlinkPath = "/usr/local/bin/quilibrium-node"

	cmd := exec.Command("sudo", "ln", "-sfn", binaryPath, symlinkPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(output))
		if msg != "" {
			return fmt.Errorf("failed to create symlink: %w\nOutput: %s", err, msg)
		}
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	linkTarget, err := os.Readlink(symlinkPath)
	if err != nil {
		return fmt.Errorf("failed to verify symlink %s: %w", symlinkPath, err)
	}
	if linkTarget != binaryPath {
		return fmt.Errorf("symlink verification failed: %s points to %s (expected %s)", symlinkPath, linkTarget, binaryPath)
	}

	return nil
}
