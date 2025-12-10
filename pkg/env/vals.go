// vals.go - Secret resolution using helmfile/vals
//
// This file adds vals integration to resolve ref+ prefixes in environment variables.
// Supports all vals backends: Vault, AWS, 1Password, file, echo, etc.
//
// Usage:
//
//	// At startup, before reading any config
//	if err := env.ResolveEnvSecrets(); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Now env vars have real values
//	password := env.GetEnv("DB_PASSWORD", "")
//
// Testing (no external dependencies):
//
//	DB_PASSWORD=ref+echo://test-password  → resolves to "test-password"
//	API_KEY=ref+file://./secrets/key.txt  → reads from file
//
// Production:
//
//	DB_PASSWORD=ref+vault://secret/db#password
//	API_KEY=ref+awssecrets://prod/api#key
//	TOKEN=ref+op://Dev/API/token  (1Password)

package env

import (
	"fmt"
	"os"
	"strings"

	"github.com/helmfile/vals"
)

const refPrefix = "ref+"

// ResolveEnvSecrets scans all environment variables for ref+ prefixes
// and resolves them using vals. The resolved values replace the original
// environment variables.
//
// This should be called early in main(), before any config is read.
//
// Example:
//
//	func main() {
//	    if err := env.ResolveEnvSecrets(); err != nil {
//	        log.Fatalf("resolving secrets: %v", err)
//	    }
//	    // Now all ref+ values are resolved
//	    cfg := env.LoadConfig()
//	}
func ResolveEnvSecrets() error {
	return ResolveEnvSecretsWithOptions(vals.Options{})
}

// ResolveEnvSecretsWithOptions resolves env secrets with custom vals options.
// Use this if you need to configure caching, logging, or AWS settings.
func ResolveEnvSecretsWithOptions(opts vals.Options) error {
	// Collect env vars that need resolution
	toResolve := make(map[string]interface{})
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		if strings.HasPrefix(value, refPrefix) {
			toResolve[key] = value
		}
	}

	// Nothing to resolve
	if len(toResolve) == 0 {
		return nil
	}

	// Create vals runtime
	runtime, err := vals.New(opts)
	if err != nil {
		return fmt.Errorf("creating vals runtime: %w", err)
	}

	// Resolve all refs
	resolved, err := runtime.Eval(toResolve)
	if err != nil {
		return fmt.Errorf("resolving secrets: %w", err)
	}

	// Update environment with resolved values
	for key, value := range resolved {
		strValue, ok := value.(string)
		if !ok {
			// Handle non-string values (shouldn't happen for env vars)
			strValue = fmt.Sprintf("%v", value)
		}
		if err := os.Setenv(key, strValue); err != nil {
			return fmt.Errorf("setting %s: %w", key, err)
		}
	}

	return nil
}

// HasSecretRefs returns true if any environment variable contains a ref+ prefix.
// Useful for checking if secret resolution is needed.
func HasSecretRefs() bool {
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[1], refPrefix) {
			return true
		}
	}
	return false
}

// ListSecretRefs returns a list of environment variable names that contain ref+ prefixes.
// Useful for debugging and logging which secrets need resolution.
func ListSecretRefs() []string {
	var refs []string
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[1], refPrefix) {
			refs = append(refs, parts[0])
		}
	}
	return refs
}

// ResolveString resolves a single value if it has a ref+ prefix.
// Returns the original value if no prefix is present.
//
// Example:
//
//	resolved, err := env.ResolveString("ref+echo://my-secret")
//	// resolved = "my-secret"
func ResolveString(value string) (string, error) {
	if !strings.HasPrefix(value, refPrefix) {
		return value, nil
	}

	runtime, err := vals.New(vals.Options{})
	if err != nil {
		return "", fmt.Errorf("creating vals runtime: %w", err)
	}

	resolved, err := runtime.Eval(map[string]interface{}{
		"value": value,
	})
	if err != nil {
		return "", fmt.Errorf("resolving %q: %w", value, err)
	}

	result, ok := resolved["value"].(string)
	if !ok {
		return fmt.Sprintf("%v", resolved["value"]), nil
	}
	return result, nil
}
