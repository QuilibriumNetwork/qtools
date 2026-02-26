package node

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// MigrateKeyData migrates keys.yml and sensitive config fields from a source
// directory into the destination node config directory.
//
// This is the Go equivalent of scripts/config/migrate-keys-data.sh
//
// It performs the following:
//  1. Copies keys.yml directly from source to destination
//  2. Reads .key.keyManagerFile.encryptionKey from source config.yml
//  3. Reads .p2p.peerPrivKey from source config.yml
//  4. Writes both values into the destination config.yml
func MigrateKeyData(sourceDir string, destDir string) error {
	sourceKeysFile := filepath.Join(sourceDir, "keys.yml")
	sourceConfigFile := filepath.Join(sourceDir, "config.yml")
	destKeysFile := filepath.Join(destDir, "keys.yml")
	destConfigFile := filepath.Join(destDir, "config.yml")

	// Validate source files exist
	if _, err := os.Stat(sourceKeysFile); os.IsNotExist(err) {
		return fmt.Errorf("source keys.yml not found at %s", sourceKeysFile)
	}
	if _, err := os.Stat(sourceConfigFile); os.IsNotExist(err) {
		return fmt.Errorf("source config.yml not found at %s", sourceConfigFile)
	}

	// Validate destination config exists
	if _, err := os.Stat(destConfigFile); os.IsNotExist(err) {
		return fmt.Errorf("destination config.yml not found at %s (create one first)", destConfigFile)
	}

	// Read source config
	sourceConfigData, err := os.ReadFile(sourceConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read source config.yml: %w", err)
	}

	var sourceConfig map[string]interface{}
	if err := yaml.Unmarshal(sourceConfigData, &sourceConfig); err != nil {
		return fmt.Errorf("failed to parse source config.yml: %w", err)
	}

	// Extract encryption key from source: .key.keyManagerFile.encryptionKey
	encryptionKey, err := extractNestedString(sourceConfig, "key", "keyManagerFile", "encryptionKey")
	if err != nil {
		return fmt.Errorf("failed to extract encryption key from source: %w", err)
	}
	if encryptionKey == "" {
		return fmt.Errorf("source .key.keyManagerFile.encryptionKey is empty")
	}

	// Extract peer private key from source: .p2p.peerPrivKey
	peerPrivKey, err := extractNestedString(sourceConfig, "p2p", "peerPrivKey")
	if err != nil {
		return fmt.Errorf("failed to extract peer private key from source: %w", err)
	}
	if peerPrivKey == "" {
		return fmt.Errorf("source .p2p.peerPrivKey is empty")
	}

	// Copy keys.yml directly (this file is always imported as-is)
	fmt.Printf("  Copying keys.yml -> %s\n", destKeysFile)
	keysData, err := os.ReadFile(sourceKeysFile)
	if err != nil {
		return fmt.Errorf("failed to read source keys.yml: %w", err)
	}
	if err := os.WriteFile(destKeysFile, keysData, 0644); err != nil {
		return fmt.Errorf("failed to write keys.yml: %w", err)
	}

	// Read destination config and update sensitive fields
	destConfigData, err := os.ReadFile(destConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read destination config.yml: %w", err)
	}

	var destConfig map[string]interface{}
	if err := yaml.Unmarshal(destConfigData, &destConfig); err != nil {
		return fmt.Errorf("failed to parse destination config.yml: %w", err)
	}

	// Set .key.keyManagerFile.encryptionKey in destination
	fmt.Println("  Migrating .key.keyManagerFile.encryptionKey")
	if err := setNestedMapValue(destConfig, encryptionKey, "key", "keyManagerFile", "encryptionKey"); err != nil {
		return fmt.Errorf("failed to set encryption key: %w", err)
	}

	// Set .p2p.peerPrivKey in destination
	fmt.Println("  Migrating .p2p.peerPrivKey")
	if err := setNestedMapValue(destConfig, peerPrivKey, "p2p", "peerPrivKey"); err != nil {
		return fmt.Errorf("failed to set peer private key: %w", err)
	}

	// Write updated destination config
	updatedData, err := yaml.Marshal(destConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	if err := os.WriteFile(destConfigFile, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated config.yml: %w", err)
	}

	fmt.Printf("  Updated %s with migrated key data\n", destConfigFile)
	return nil
}

// extractNestedString extracts a string value from a nested map using a sequence of keys
func extractNestedString(m map[string]interface{}, keys ...string) (string, error) {
	current := m
	for i, key := range keys {
		val, ok := current[key]
		if !ok {
			return "", fmt.Errorf("key %q not found at depth %d", key, i)
		}

		if i == len(keys)-1 {
			// Last key - expect a string value
			str, ok := val.(string)
			if !ok {
				return "", fmt.Errorf("value at %q is not a string (got %T)", key, val)
			}
			return str, nil
		}

		// Intermediate key - expect a map
		next, ok := val.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("value at %q is not a map (got %T)", key, val)
		}
		current = next
	}

	return "", fmt.Errorf("empty key path")
}

// setNestedMapValue sets a value in a nested map, creating intermediate maps as needed
func setNestedMapValue(m map[string]interface{}, value interface{}, keys ...string) error {
	if len(keys) == 0 {
		return fmt.Errorf("empty key path")
	}

	current := m
	for i, key := range keys {
		if i == len(keys)-1 {
			// Last key - set the value
			current[key] = value
			return nil
		}

		// Intermediate key - navigate or create map
		val, ok := current[key]
		if !ok {
			// Create new map
			newMap := make(map[string]interface{})
			current[key] = newMap
			current = newMap
			continue
		}

		next, ok := val.(map[string]interface{})
		if !ok {
			// Overwrite non-map value with a new map
			newMap := make(map[string]interface{})
			current[key] = newMap
			current = newMap
			continue
		}
		current = next
	}

	return nil
}
