// Example: Using helmfile/vals for secret resolution
//
// vals resolves secrets from 25+ backends using URI syntax:
//   ref+echo://value           - returns "value" (for testing)
//   ref+file:///path/to/file   - reads from file
//   ref+vault://path#key       - HashiCorp Vault
//   ref+awssecrets://name#key  - AWS Secrets Manager
//   ref+1password://vault/item#field - 1Password
//
// Run:
//   go run main.go
//   SECRET=ref+echo://my-secret-value go run main.go
//   SECRET=ref+file://./testdata/secret.txt go run main.go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/helmfile/vals"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Create vals runtime
	runtime, err := vals.New(vals.Options{})
	if err != nil {
		return fmt.Errorf("creating vals runtime: %w", err)
	}

	ctx := context.Background()

	fmt.Println("vals Secret Resolution Examples")
	fmt.Println("================================")
	fmt.Println()

	// Example 1: Echo provider (built-in, great for testing)
	// Returns the path as the value
	echoRef := "ref+echo://my-secret-password"
	echoVal, err := runtime.Eval(map[string]interface{}{
		"secret": echoRef,
	})
	if err != nil {
		return fmt.Errorf("echo eval: %w", err)
	}
	fmt.Printf("1. Echo provider:\n")
	fmt.Printf("   Input:  %s\n", echoRef)
	fmt.Printf("   Output: %v\n\n", echoVal["secret"])

	// Example 2: Resolve from environment variable
	// If SECRET env var contains a ref+, resolve it
	secretEnv := os.Getenv("SECRET")
	if secretEnv == "" {
		secretEnv = "ref+echo://default-from-echo"
	}
	fmt.Printf("2. From SECRET env var:\n")
	fmt.Printf("   Raw:    %s\n", secretEnv)

	if isRef(secretEnv) {
		resolved, err := runtime.Eval(map[string]interface{}{
			"value": secretEnv,
		})
		if err != nil {
			return fmt.Errorf("resolving SECRET: %w", err)
		}
		fmt.Printf("   Resolved: %v\n\n", resolved["value"])
	} else {
		fmt.Printf("   (not a ref, using as-is): %s\n\n", secretEnv)
	}

	// Example 3: Batch resolve multiple secrets
	fmt.Printf("3. Batch resolution:\n")
	secrets := map[string]interface{}{
		"db_password":  "ref+echo://super-secret-db-pass",
		"api_key":      "ref+echo://api-key-12345",
		"plain_value":  "not-a-ref-just-a-string",
	}

	resolved, err := runtime.Eval(secrets)
	if err != nil {
		return fmt.Errorf("batch eval: %w", err)
	}

	for k, v := range resolved {
		fmt.Printf("   %s = %v\n", k, v)
	}
	fmt.Println()

	// Example 4: File provider (if testdata exists)
	testdataPath := "./testdata/secret.txt"
	if _, err := os.Stat(testdataPath); err == nil {
		fileRef := "ref+file://" + testdataPath
		fileResolved, err := runtime.Eval(map[string]interface{}{
			"from_file": fileRef,
		})
		if err != nil {
			return fmt.Errorf("file eval: %w", err)
		}
		fmt.Printf("4. File provider:\n")
		fmt.Printf("   Input:  %s\n", fileRef)
		fmt.Printf("   Output: %v\n\n", fileResolved["from_file"])
	} else {
		fmt.Printf("4. File provider: (skipped - create %s to test)\n\n", testdataPath)
	}

	// Example 5: Show how to scan env vars for refs
	fmt.Printf("5. Scanning environment for ref+ patterns:\n")
	refCount := 0
	for _, env := range os.Environ() {
		for i := 0; i < len(env); i++ {
			if env[i] == '=' {
				value := env[i+1:]
				if isRef(value) {
					name := env[:i]
					fmt.Printf("   Found: %s\n", name)
					refCount++
				}
				break
			}
		}
	}
	if refCount == 0 {
		fmt.Printf("   (none found - try: SECRET=ref+echo://test go run main.go)\n")
	}
	fmt.Println()

	// Example 6: Using Get for single value resolution
	fmt.Printf("6. Single value resolution with Get():\n")
	_ = ctx // ctx available if needed by other providers
	singleVal, err := runtime.Get("ref+echo://single-value")
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	fmt.Printf("   Result: %s\n", singleVal)

	return nil
}

// isRef checks if a string is a vals reference
func isRef(s string) bool {
	return len(s) > 4 && s[:4] == "ref+"
}
