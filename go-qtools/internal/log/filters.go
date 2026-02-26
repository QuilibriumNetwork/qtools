package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LogFilter represents a log filter configuration
type LogFilter struct {
	Mode    string            `yaml:"mode"`    // "include" or "exclude"
	Filters map[string]bool   `yaml:"filters"` // Map of filter string to enabled state
}

// FilterLogLine applies the filter to a log line
// Returns true if the line should be shown, false if it should be hidden
func FilterLogLine(line string, filter *LogFilter) bool {
	if filter == nil || len(filter.Filters) == 0 {
		return true // No filter, show all lines
	}

	lineLower := strings.ToLower(line)
	hasMatch := false

	// Check each active filter
	for filterStr, enabled := range filter.Filters {
		if !enabled {
			continue
		}

		if strings.Contains(lineLower, strings.ToLower(filterStr)) {
			hasMatch = true
			break
		}
	}

	// Apply mode logic
	if filter.Mode == "include" {
		// Include mode: show only lines that match
		return hasMatch
	} else if filter.Mode == "exclude" {
		// Exclude mode: hide lines that match
		return !hasMatch
	}

	// Default: show all
	return true
}

// LoadFilters loads filters from a YAML file
func LoadFilters(path string) (*LogFilter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty filter if file doesn't exist
			return &LogFilter{
				Mode:    "include",
				Filters: make(map[string]bool),
			}, nil
		}
		return nil, fmt.Errorf("failed to read filter file: %w", err)
	}

	var filter LogFilter
	if err := yaml.Unmarshal(data, &filter); err != nil {
		return nil, fmt.Errorf("failed to parse filter file: %w", err)
	}

	if filter.Filters == nil {
		filter.Filters = make(map[string]bool)
	}

	return &filter, nil
}

// SaveFilters saves filters to a YAML file
func SaveFilters(filter *LogFilter, path string) error {
	if filter == nil {
		return fmt.Errorf("filter is nil")
	}

	data, err := yaml.Marshal(filter)
	if err != nil {
		return fmt.Errorf("failed to marshal filter: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create filter directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write filter file: %w", err)
	}

	return nil
}

// GetFilterPath returns the default path for log filter file
func GetFilterPath() string {
	// Default to $QTOOLS_PATH/log-selector-list.yml
	qtoolsPath := os.Getenv("QTOOLS_PATH")
	if qtoolsPath == "" {
		qtoolsPath = "/home/quilibrium/qtools"
	}
	return fmt.Sprintf("%s/log-selector-list.yml", qtoolsPath)
}
