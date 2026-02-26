package config

import (
	"fmt"
	"runtime"
	"strings"
)

// GenerateDefaultConfig generates a default config programmatically
func GenerateDefaultConfig() *Config {
	config := &Config{
		Raw: make(map[string]interface{}),
		User: "ubuntu",
		QuilibriumRepoDir: ".",
		ReleaseVersion: "1.4.19.1",
		CurrentNodeVersion: "1.4.19.1",
		CurrentQClientVersion: "1.4.19.1",
		OSArch: getOSArch(),
		QtoolsVersion: 28,
		QClientCLIName: "qclient",
		SSH: &SSHConfig{
			AllowFromIP: false,
			Port: 22,
			Skip192168Block: false,
		},
		Service: &ServiceConfig{
			FileName: "ceremonyclient",
			Debug: false,
			SignatureCheck: false,
			Testnet: false,
			WorkingDir: "/home/quilibrium/node",
			LinkDirectory: "/usr/local/bin",
			LinkName: "quilibrium-node",
			DefaultUser: "quilibrium",
			QuilibriumNodePath: "/home/quilibrium/node",
			QuilibriumClientPath: "/home/quilibrium/client",
			RestartTime: "5s",
			WorkerService: &WorkerServiceConfig{
				GOGC: "",
				GOMEMLimit: "",
				RestartTime: "5s",
			},
			Clustering: &ClusteringConfig{
				Enabled: false,
				MasterServiceName: "ceremonyclient",
				LocalOnly: false,
				DataWorkerServiceName: "dataworker",
				WorkerBaseP2PPort: 25000,
				WorkerBaseStreamPort: 32500,
				MasterStreamPort: 8340,
				DefaultSSHPort: 22,
				DefaultUser: "ubuntu",
				SSHKeyPath: "$HOME/.ssh",
				DataWorkerPriority: 90,
				SSHKeyName: "cluster-key",
				MainIP: "",
				Servers: []string{},
				AutoRemovedServers: []string{},
			},
			Args: "",
			MaxThreads: false,
		},
		DataWorkerService: &DataWorkerServiceConfig{
			WorkerCount: 0,
			BaseIndex: 1,
		},
		Manual: &ManualConfig{
			Enabled: true, // Opinionated default: manual mode for better reliability
			WorkerCount: 0, // Auto-calculated based on CPU cores
			LocalOnly: true,
		},
		ScheduledTasks: &ScheduledTasksConfig{},
		Settings: &SettingsConfig{
			UseAVX512: false,
			LogFile: "debug.log",
			InternalIP: "",
		},
		Dev: &DevConfig{
			DefaultRepoBranch: "develop",
			DefaultRepoURL: "https://github.com/tjsturos/ceremonyclient.git",
			DefaultRepoPath: "$HOME/quil-dev",
		},
		NodeRegistry: &NodeRegistry{
			Nodes: []RegisteredNode{},
		},
		QStorage: &QStorageConfig{
			AccessKeyID: "",
			AccessKey:   "",
			AccountID:   "",
			Bucket:      "",
			Region:      "q-world-1",
			EndpointURL: "https://qstorage.quilibrium.com",
			Prefix:      "",
		},
	}

	// Set config version
	config.Raw = make(map[string]interface{})
	config.Raw["config_version"] = "1.4"

	return config
}

// MergeDefaults merges user config with defaults
func MergeDefaults(config *Config) *Config {
	defaults := GenerateDefaultConfig()

	// Merge service config
	if config.Service == nil {
		config.Service = defaults.Service
	} else {
		if config.Service.WorkerService == nil {
			config.Service.WorkerService = defaults.Service.WorkerService
		}
		if config.Service.Clustering == nil {
			config.Service.Clustering = defaults.Service.Clustering
		}
	}

	// Merge manual config
	if config.Manual == nil {
		config.Manual = defaults.Manual
	} else {
		// Ensure manual mode defaults to enabled if not explicitly set
		// This is an opinionated default for better reliability
		if !config.Manual.Enabled && !isExplicitlySet(config.Raw, "manual.enabled") {
			config.Manual.Enabled = true
		}
	}

	// Merge other configs
	if config.SSH == nil {
		config.SSH = defaults.SSH
	}
	if config.DataWorkerService == nil {
		config.DataWorkerService = defaults.DataWorkerService
	}
	if config.ScheduledTasks == nil {
		config.ScheduledTasks = defaults.ScheduledTasks
	}
	if config.Settings == nil {
		config.Settings = defaults.Settings
	}
	if config.Dev == nil {
		config.Dev = defaults.Dev
	}
	if config.NodeRegistry == nil {
		config.NodeRegistry = defaults.NodeRegistry
	}
	if config.QStorage == nil {
		config.QStorage = defaults.QStorage
	} else {
		// Ensure defaults for region and endpoint if not explicitly set
		if config.QStorage.Region == "" {
			config.QStorage.Region = defaults.QStorage.Region
		}
		if config.QStorage.EndpointURL == "" {
			config.QStorage.EndpointURL = defaults.QStorage.EndpointURL
		}
	}

	return config
}

// GetConfigValue gets a config value by dot-separated path (e.g., "scheduled_tasks.status.enabled")
func GetConfigValue(config *Config, path string) (interface{}, error) {
	if config.Raw == nil {
		return nil, fmt.Errorf("raw config is nil")
	}

	keys := strings.Split(path, ".")
	current := config.Raw

	for i, key := range keys {
		if i == len(keys)-1 {
			// Last key, return value
			if val, ok := current[key]; ok {
				return val, nil
			}
			return nil, fmt.Errorf("key %s not found", path)
		}

		// Navigate deeper
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			return nil, fmt.Errorf("path %s is not a map at key %s", path, key)
		}
	}

	return nil, fmt.Errorf("key %s not found", path)
}

// SetConfigValue sets a config value by dot-separated path
func SetConfigValue(config *Config, path string, value interface{}) error {
	if config.Raw == nil {
		config.Raw = make(map[string]interface{})
	}

	keys := strings.Split(path, ".")
	current := config.Raw

	for i, key := range keys {
		if i == len(keys)-1 {
			// Last key, set value
			current[key] = value
			return nil
		}

		// Navigate deeper, creating maps as needed
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			// Create new map
			current[key] = make(map[string]interface{})
			current = current[key].(map[string]interface{})
		}
	}

	return nil
}

// isExplicitlySet checks if a config value was explicitly set (not just default)
func isExplicitlySet(raw map[string]interface{}, path string) bool {
	if raw == nil {
		return false
	}

	keys := strings.Split(path, ".")
	current := raw

	for i, key := range keys {
		if i == len(keys)-1 {
			_, exists := current[key]
			return exists
		}

		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			return false
		}
	}

	return false
}

// getOSArch returns the OS architecture string
func getOSArch() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go arch to expected format
	archMap := map[string]string{
		"amd64": "amd64",
		"arm64": "arm64",
		"386":   "386",
	}

	mappedArch := archMap[arch]
	if mappedArch == "" {
		mappedArch = arch
	}

	return fmt.Sprintf("%s-%s", os, mappedArch)
}
