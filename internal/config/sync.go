package config

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// SyncResult describes the outcome of a config sync operation.
type SyncResult struct {
	ConfigPath string   // Path to the config file
	Added      []string // Keys added as commented lines
	Removed    []string // Deprecated keys removed
	Preserved  int      // Count of preserved user values
	DryRun     bool
	Changed    bool // True if any changes were made/would be made
}

// SyncOptions configures how config sync behaves.
type SyncOptions struct {
	DryRun bool // Preview changes without writing
}

// flattenDefaults converts the nested GetDefaults() map to dot-notation keys.
// Example: {"notifications": {"enabled": true}} -> {"notifications.enabled": true}
func flattenDefaults() map[string]interface{} {
	result := make(map[string]interface{})
	flattenMap("", GetDefaults(), result)
	return result
}

// flattenMap recursively flattens a nested map into dot-notation keys.
func flattenMap(prefix string, m map[string]interface{}, result map[string]interface{}) {
	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// Recurse into nested maps
			flattenMap(fullKey, v, result)
		default:
			// Leaf value
			result[fullKey] = value
		}
	}
}

// extractUserKeys walks the YAML node tree and returns all keys in dot-notation.
func extractUserKeys(node *yaml.Node) []string {
	var keys []string
	extractKeysFromNode("", node, &keys)
	return keys
}

// extractKeysFromNode recursively extracts keys from a YAML node.
func extractKeysFromNode(prefix string, node *yaml.Node, keys *[]string) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		// Process document content
		for _, child := range node.Content {
			extractKeysFromNode(prefix, child, keys)
		}
	case yaml.MappingNode:
		// Process key-value pairs
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			fullKey := keyNode.Value
			if prefix != "" {
				fullKey = prefix + "." + keyNode.Value
			}

			// Check if value is a nested map
			if valueNode.Kind == yaml.MappingNode {
				extractKeysFromNode(fullKey, valueNode, keys)
			} else {
				*keys = append(*keys, fullKey)
			}
		}
	}
}

// findMissingKeys returns keys that exist in schema but not in user config.
func findMissingKeys(userKeys []string, schemaKeys map[string]interface{}) []string {
	userSet := make(map[string]bool, len(userKeys))
	for _, k := range userKeys {
		userSet[k] = true
	}

	var missing []string
	for schemaKey := range schemaKeys {
		if !userSet[schemaKey] {
			missing = append(missing, schemaKey)
		}
	}

	sort.Strings(missing)
	return missing
}

// findDeprecatedKeys returns keys that exist in user config but not in schema.
func findDeprecatedKeys(userKeys []string, schemaKeys map[string]interface{}) []string {
	var deprecated []string
	for _, userKey := range userKeys {
		if _, exists := schemaKeys[userKey]; !exists {
			deprecated = append(deprecated, userKey)
		}
	}

	sort.Strings(deprecated)
	return deprecated
}

// generateNewKeysBlock creates a YAML block for missing keys with their defaults.
func generateNewKeysBlock(missing []string, defaults map[string]interface{}) string {
	if len(missing) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n# New configuration options (added by autospec config sync)\n")

	for _, key := range missing {
		value, ok := defaults[key]
		if !ok {
			continue
		}

		// Format the value appropriately
		formattedValue := formatValue(value)

		// Get description from KnownKeys if available
		if schema, exists := KnownKeys[key]; exists && schema.Description != "" {
			sb.WriteString(fmt.Sprintf("# %s\n", schema.Description))
		}

		// Write the key-value pair (NOT commented - these are new defaults!)
		sb.WriteString(fmt.Sprintf("%s: %s\n", key, formattedValue))
	}

	return sb.String()
}

// formatValue formats a value for YAML output.
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		if v == "" {
			return `""`
		}
		// Quote strings that might be interpreted as other types
		if strings.ContainsAny(v, ": #[]{}!&*?|>'\"\n\\") {
			return fmt.Sprintf("%q", v)
		}
		return v
	case bool:
		return fmt.Sprintf("%v", v)
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	case []string:
		if len(v) == 0 {
			return "[]"
		}
		return fmt.Sprintf("[%s]", strings.Join(v, ", "))
	case []interface{}:
		if len(v) == 0 {
			return "[]"
		}
		var items []string
		for _, item := range v {
			items = append(items, formatValue(item))
		}
		return fmt.Sprintf("[%s]", strings.Join(items, ", "))
	default:
		return fmt.Sprintf("%v", v)
	}
}

// removeDeprecatedKeys removes deprecated keys from the YAML node tree.
func removeDeprecatedKeys(node *yaml.Node, deprecated []string) error {
	if len(deprecated) == 0 {
		return nil
	}

	deprecatedSet := make(map[string]bool, len(deprecated))
	for _, k := range deprecated {
		deprecatedSet[k] = true
	}

	return removeKeysFromNode("", node, deprecatedSet)
}

// removeKeysFromNode recursively removes keys from a YAML node.
func removeKeysFromNode(prefix string, node *yaml.Node, deprecated map[string]bool) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			if err := removeKeysFromNode(prefix, child, deprecated); err != nil {
				return err
			}
		}
	case yaml.MappingNode:
		// Build new content without deprecated keys
		var newContent []*yaml.Node
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			fullKey := keyNode.Value
			if prefix != "" {
				fullKey = prefix + "." + keyNode.Value
			}

			// Check if this key or any nested key is deprecated
			if deprecated[fullKey] {
				// Skip this key-value pair
				continue
			}

			// Recurse into nested maps
			if valueNode.Kind == yaml.MappingNode {
				if err := removeKeysFromNode(fullKey, valueNode, deprecated); err != nil {
					return err
				}
			}

			newContent = append(newContent, keyNode, valueNode)
		}
		node.Content = newContent
	}

	return nil
}

// SyncConfig synchronizes a config file with the current schema.
// It adds missing keys as commented lines and removes deprecated keys.
func SyncConfig(configPath string, opts SyncOptions) (*SyncResult, error) {
	result := &SyncResult{
		ConfigPath: configPath,
		DryRun:     opts.DryRun,
	}

	// Check if file exists
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file - nothing to sync
			return result, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Parse YAML
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing config YAML: %w", err)
	}

	// Get schema and user keys
	schemaKeys := flattenDefaults()
	userKeys := extractUserKeys(&root)

	// Find differences
	result.Added = findMissingKeys(userKeys, schemaKeys)
	result.Removed = findDeprecatedKeys(userKeys, schemaKeys)
	result.Preserved = len(userKeys) - len(result.Removed)
	result.Changed = len(result.Added) > 0 || len(result.Removed) > 0

	// If dry run or no changes, return early
	if opts.DryRun || !result.Changed {
		return result, nil
	}

	// Apply changes: remove deprecated keys
	if err := removeDeprecatedKeys(&root, result.Removed); err != nil {
		return nil, fmt.Errorf("removing deprecated keys: %w", err)
	}

	// Marshal back to YAML
	content, err := yaml.Marshal(&root)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	// Append block for new keys (with their default values)
	if len(result.Added) > 0 {
		newBlock := generateNewKeysBlock(result.Added, schemaKeys)
		content = append(content, []byte(newBlock)...)
	}

	// Write atomically
	if err := writeAtomically(configPath, content); err != nil {
		return nil, fmt.Errorf("writing config: %w", err)
	}

	return result, nil
}
