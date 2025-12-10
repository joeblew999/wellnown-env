// fields.go: Extract field information from config structs via reflection
//
// This analyzes structs with ardanlabs/conf tags to extract:
// - Field paths (DB.Password)
// - Types (string, int, etc.)
// - Env var names (APP_DB_PASSWORD)
// - Defaults, required flags, mask (secret) flags
// - Service dependencies (service:org/repo)
package env

import (
	"reflect"
	"strings"

	"github.com/joeblew999/wellnown-env/pkg/env/registry"
)

// ExtractFields extracts FieldInfo from a config struct using reflection.
// The prefix is the env var prefix (e.g., "APP").
func ExtractFields(prefix string, cfg interface{}) []registry.FieldInfo {
	var fields []registry.FieldInfo
	extractFieldsRecursive(prefix, "", reflect.TypeOf(cfg), &fields)
	return fields
}

func extractFieldsRecursive(prefix, path string, t reflect.Type, fields *[]registry.FieldInfo) {
	// Dereference pointers
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Build field path
		fieldPath := field.Name
		if path != "" {
			fieldPath = path + "." + field.Name
		}

		// Handle embedded structs (like conf.Version)
		if field.Anonymous {
			extractFieldsRecursive(prefix, path, field.Type, fields)
			continue
		}

		// Handle nested structs
		if field.Type.Kind() == reflect.Struct && field.Tag.Get("conf") == "" {
			extractFieldsRecursive(prefix, fieldPath, field.Type, fields)
			continue
		}

		// Parse conf tag
		confTag := field.Tag.Get("conf")
		fi := parseConfTag(prefix, fieldPath, field.Type.String(), confTag)

		*fields = append(*fields, fi)
	}
}

// parseConfTag parses an ardanlabs/conf tag and returns FieldInfo
func parseConfTag(prefix, path, typeName, tag string) registry.FieldInfo {
	fi := registry.FieldInfo{
		Path:   path,
		Type:   typeName,
		EnvKey: buildEnvKey(prefix, path),
	}

	if tag == "" {
		return fi
	}

	// Parse tag parts (comma-separated)
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		switch {
		case strings.HasPrefix(part, "default:"):
			fi.Default = strings.TrimPrefix(part, "default:")

		case strings.HasPrefix(part, "env:"):
			// Custom env var name overrides default
			fi.EnvKey = strings.TrimPrefix(part, "env:")

		case strings.HasPrefix(part, "service:"):
			// Service dependency: service:org/repo
			fi.Dependency = strings.TrimPrefix(part, "service:")

		case part == "required":
			fi.Required = true

		case part == "mask":
			fi.IsSecret = true

		case part == "noprint":
			// noprint also means secret
			fi.IsSecret = true
		}
	}

	return fi
}

// buildEnvKey converts a field path to env var name
// e.g., prefix="APP", path="DB.Password" -> "APP_DB_PASSWORD"
func buildEnvKey(prefix, path string) string {
	// Replace dots with underscores and uppercase
	key := strings.ReplaceAll(path, ".", "_")
	key = strings.ToUpper(key)

	if prefix != "" {
		key = strings.ToUpper(prefix) + "_" + key
	}

	return key
}

// GetDependencies extracts service dependencies from fields
func GetDependencies(fields []registry.FieldInfo) []string {
	var deps []string
	seen := make(map[string]bool)

	for _, f := range fields {
		if f.Dependency != "" && !seen[f.Dependency] {
			deps = append(deps, f.Dependency)
			seen[f.Dependency] = true
		}
	}

	return deps
}

// GetSecrets returns fields that are marked as secrets
func GetSecrets(fields []registry.FieldInfo) []registry.FieldInfo {
	var secrets []registry.FieldInfo
	for _, f := range fields {
		if f.IsSecret {
			secrets = append(secrets, f)
		}
	}
	return secrets
}

// GetRequired returns fields that are required
func GetRequired(fields []registry.FieldInfo) []registry.FieldInfo {
	var required []registry.FieldInfo
	for _, f := range fields {
		if f.Required {
			required = append(required, f)
		}
	}
	return required
}
