package log

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/quilibrium/qtools/go-qtools/internal/config"
	"github.com/quilibrium/qtools/go-qtools/internal/node"
)

// LogViewer handles log file tailing and viewing
type LogViewer struct {
	config *config.Config
}

// NewLogViewer creates a new log viewer
func NewLogViewer(cfg *config.Config) *LogViewer {
	return &LogViewer{
		config: cfg,
	}
}

// GetLogFilePaths gets the paths for master, worker, and qtools log files
func (lv *LogViewer) GetLogFilePaths() (masterPath string, workerPaths []string, qtoolsPath string, err error) {
	nodePath := config.GetNodePath()
	qtoolsPathEnv := config.GetQtoolsPath()

	// Check if custom logging is enabled
	nodeConfigPath := config.GetNodeConfigPath()
	loggingConfig, err := node.GetLoggingConfig(nodeConfigPath)
	if err == nil && loggingConfig != nil {
		// Custom logging is enabled
		logPath := loggingConfig.Path
		if !filepath.IsAbs(logPath) {
			logPath = filepath.Join(nodePath, logPath)
		}

		masterPath = filepath.Join(logPath, "master.log")

		// Get worker count to determine worker log paths
		workerCount := node.GetWorkerCount(lv.config)
		for i := 1; i <= workerCount; i++ {
			workerPaths = append(workerPaths, filepath.Join(logPath, fmt.Sprintf("worker-%d.log", i)))
		}
	} else {
		// Custom logging disabled, will use journalctl
		masterPath = "" // Indicates journalctl should be used
	}

	// Qtools log path
	qtoolsPath = filepath.Join(qtoolsPathEnv, "qtools.log")
	if _, err := os.Stat(qtoolsPath); os.IsNotExist(err) {
		qtoolsPath = filepath.Join(qtoolsPathEnv, "log")
	}

	return masterPath, workerPaths, qtoolsPath, nil
}

// TailLogFile tails a log file and sends lines through a channel
func (lv *LogViewer) TailLogFile(path string, filter *LogFilter) (<-chan string, error) {
	ch := make(chan string, 100)

	// Check if we should use journalctl
	if path == "" || !fileExists(path) {
		// Use journalctl
		return lv.tailJournalctl(filter)
	}

	// Use tail -F for file
	go func() {
		defer close(ch)

		cmd := exec.Command("tail", "-F", path)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return
		}

		if err := cmd.Start(); err != nil {
			return
		}
		defer cmd.Process.Kill()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if FilterLogLine(line, filter) {
				ch <- line
			}
		}
	}()

	return ch, nil
}

// tailJournalctl tails journalctl output
func (lv *LogViewer) tailJournalctl(filter *LogFilter) (<-chan string, error) {
	ch := make(chan string, 100)

	serviceName := "ceremonyclient"
	if lv.config != nil && lv.config.Service != nil && lv.config.Service.FileName != "" {
		serviceName = lv.config.Service.FileName
	}

	go func() {
		defer close(ch)

		cmd := exec.Command("sudo", "journalctl", "-u", serviceName, "-f", "--no-hostname", "-o", "cat")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return
		}

		if err := cmd.Start(); err != nil {
			return
		}
		defer cmd.Process.Kill()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if FilterLogLine(line, filter) {
				ch <- line
			}
		}
	}()

	return ch, nil
}

// TailWorkerLog tails a specific worker log
func (lv *LogViewer) TailWorkerLog(workerIndex int, filter *LogFilter) (<-chan string, error) {
	_, workerPaths, _, err := lv.GetLogFilePaths()
	if err != nil {
		return nil, err
	}

	if workerIndex < 1 || workerIndex > len(workerPaths) {
		return nil, fmt.Errorf("invalid worker index: %d", workerIndex)
	}

	return lv.TailLogFile(workerPaths[workerIndex-1], filter)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ReadLogLines reads the last N lines from a log file
func (lv *LogViewer) ReadLogLines(path string, lines int, filter *LogFilter) ([]string, error) {
	if path == "" || !fileExists(path) {
		return []string{}, nil // Can't read from journalctl history easily
	}

	cmd := exec.Command("tail", "-n", fmt.Sprintf("%d", lines), path)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var result []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if FilterLogLine(line, filter) {
			result = append(result, line)
		}
	}

	return result, nil
}
