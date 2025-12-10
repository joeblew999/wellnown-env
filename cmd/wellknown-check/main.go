// wellknown-check: CI/CD configuration validation tool
//
// This CLI tool helps validate service configurations:
// - Export service schema as JSON
// - Check if dependencies are registered
// - Analyze impact on consumers
//
// Usage:
//
//	wellknown-check --schema-dump           # Output schema as JSON
//	wellknown-check --check-deps            # Check dependency availability
//	wellknown-check --check-consumers       # Check impact on consumers
//	wellknown-check --self                  # Show changes in this service
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/joeblew999/wellnown-env/pkg/env"
	"github.com/joeblew999/wellnown-env/pkg/env/registry"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Define flags
	schemaDump := flag.Bool("schema-dump", false, "Output service schema as JSON")
	checkDeps := flag.Bool("check-deps", false, "Check if dependencies are available in NATS registry")
	checkConsumers := flag.Bool("check-consumers", false, "Check impact on services that depend on this service")
	selfCheck := flag.Bool("self", false, "Show local changes in this service's config requirements")
	prSchema := flag.String("pr-schema", "", "Path to PR schema file for comparison")
	repo := flag.String("repo", "", "Repository name (org/repo) for this service")
	timeout := flag.Duration("timeout", 10*time.Second, "Timeout for NATS operations")

	flag.Parse()

	// At least one action required
	if !*schemaDump && !*checkDeps && !*checkConsumers && !*selfCheck {
		flag.Usage()
		return fmt.Errorf("at least one action flag required")
	}

	// Create manager with minimal options (no GUI, no heartbeat)
	mgr, err := env.New("WELLKNOWN_CHECK",
		env.WithoutGUI(),
		env.WithoutHeartbeat(),
		env.WithoutRegistration(),
	)
	if err != nil {
		return fmt.Errorf("creating manager: %w", err)
	}
	defer mgr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Handle schema dump
	if *schemaDump {
		return dumpSchema(mgr, *repo)
	}

	// Handle self check
	if *selfCheck {
		return selfCheckChanges(mgr, *prSchema)
	}

	// Handle dependency check
	if *checkDeps {
		return checkDependencies(ctx, mgr)
	}

	// Handle consumer check
	if *checkConsumers {
		return checkConsumerImpact(ctx, mgr, *repo)
	}

	return nil
}

// dumpSchema outputs the service schema as JSON
func dumpSchema(mgr *env.Manager, repo string) error {
	// Get schema from manager's registration
	reg := mgr.Registration()
	if reg == nil {
		return fmt.Errorf("no registration available (service not configured)")
	}

	// If repo is provided, override GitHub identity
	if repo != "" {
		// Parse org/repo format
		for i, c := range repo {
			if c == '/' {
				reg.GitHub.Org = repo[:i]
				reg.GitHub.Repo = repo[i+1:]
				break
			}
		}
	}

	// Output as JSON
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(reg)
}

// selfCheckChanges shows local changes in this service's config
func selfCheckChanges(mgr *env.Manager, prSchemaPath string) error {
	reg := mgr.Registration()
	if reg == nil {
		return fmt.Errorf("no registration available")
	}

	fmt.Printf("Service: %s/%s\n", reg.GitHub.Org, reg.GitHub.Repo)
	fmt.Printf("Instance: %s\n", reg.Instance.ID)
	fmt.Println()

	if prSchemaPath != "" {
		// Compare with PR schema file
		prData, err := os.ReadFile(prSchemaPath)
		if err != nil {
			return fmt.Errorf("reading PR schema: %w", err)
		}

		var prReg registry.ServiceRegistration
		if err := json.Unmarshal(prData, &prReg); err != nil {
			return fmt.Errorf("parsing PR schema: %w", err)
		}

		// Compare fields
		fmt.Println("Changes from PR:")
		compareFields(reg.Fields, prReg.Fields)
	} else {
		// Just show current fields
		fmt.Println("Current configuration fields:")
		for _, f := range reg.Fields {
			required := ""
			if f.Required {
				required = " (required)"
			}
			secret := ""
			if f.IsSecret {
				secret = " [secret]"
			}
			dep := ""
			if f.Dependency != "" {
				dep = fmt.Sprintf(" -> %s", f.Dependency)
			}
			fmt.Printf("  %s: %s%s%s%s\n", f.EnvKey, f.Type, required, secret, dep)
		}
	}

	return nil
}

// compareFields compares two sets of fields and prints differences
func compareFields(current, pr []registry.FieldInfo) {
	currentMap := make(map[string]registry.FieldInfo)
	for _, f := range current {
		currentMap[f.EnvKey] = f
	}

	prMap := make(map[string]registry.FieldInfo)
	for _, f := range pr {
		prMap[f.EnvKey] = f
	}

	// Check for added fields
	for key, f := range currentMap {
		if _, exists := prMap[key]; !exists {
			fmt.Printf("  + %s (new", key)
			if f.Required {
				fmt.Print(", required")
			}
			fmt.Println(")")
		}
	}

	// Check for removed fields
	for key := range prMap {
		if _, exists := currentMap[key]; !exists {
			fmt.Printf("  - %s (removed)\n", key)
		}
	}

	// Check for modified fields
	for key, curr := range currentMap {
		if pr, exists := prMap[key]; exists {
			changes := []string{}
			if curr.Default != pr.Default {
				changes = append(changes, fmt.Sprintf("default: %s -> %s", pr.Default, curr.Default))
			}
			if curr.Required != pr.Required {
				if curr.Required {
					changes = append(changes, "now required")
				} else {
					changes = append(changes, "no longer required")
				}
			}
			if curr.IsSecret != pr.IsSecret {
				if curr.IsSecret {
					changes = append(changes, "now secret")
				} else {
					changes = append(changes, "no longer secret")
				}
			}
			if len(changes) > 0 {
				fmt.Printf("  ~ %s: %v\n", key, changes)
			}
		}
	}
}

// checkDependencies checks if dependencies are available in NATS registry
func checkDependencies(ctx context.Context, mgr *env.Manager) error {
	reg := mgr.Registration()
	if reg == nil {
		return fmt.Errorf("no registration available")
	}

	kv := mgr.KV()
	if kv == nil {
		return fmt.Errorf("NATS KV not available (not connected to hub?)")
	}

	deps := env.GetDependencies(reg.Fields)
	if len(deps) == 0 {
		fmt.Println("No dependencies declared.")
		return nil
	}

	fmt.Printf("Checking %d dependencies:\n", len(deps))
	allFound := true

	for _, dep := range deps {
		exists, err := env.ServiceExists(ctx, kv, dep)
		if err != nil {
			fmt.Printf("  ! %s: error checking (%v)\n", dep, err)
			allFound = false
		} else if exists {
			fmt.Printf("  ✓ %s: available\n", dep)
		} else {
			fmt.Printf("  ✗ %s: not found\n", dep)
			allFound = false
		}
	}

	if !allFound {
		return fmt.Errorf("some dependencies not available")
	}
	return nil
}

// checkConsumerImpact checks impact on services that depend on this service
func checkConsumerImpact(ctx context.Context, mgr *env.Manager, repo string) error {
	kv := mgr.KV()
	if kv == nil {
		return fmt.Errorf("NATS KV not available (not connected to hub?)")
	}

	// Determine this service's identity
	thisService := repo
	if thisService == "" {
		reg := mgr.Registration()
		if reg != nil && reg.GitHub.Org != "" {
			thisService = reg.GitHub.Org + "/" + reg.GitHub.Repo
		}
	}
	if thisService == "" {
		return fmt.Errorf("service identity required (use --repo flag or set GitOrg/GitRepo)")
	}

	fmt.Printf("Checking consumers of %s:\n", thisService)

	// Get all services
	services, err := env.GetAllServices(ctx, kv)
	if err != nil {
		return fmt.Errorf("fetching services: %w", err)
	}

	consumers := 0
	for _, svc := range services {
		deps := env.GetDependencies(svc.Fields)
		for _, dep := range deps {
			if dep == thisService {
				consumers++
				fmt.Printf("  • %s/%s depends on this service\n", svc.GitHub.Org, svc.GitHub.Repo)
				break
			}
		}
	}

	if consumers == 0 {
		fmt.Println("  No consumers found.")
	} else {
		fmt.Printf("\n%d service(s) depend on %s\n", consumers, thisService)
	}

	return nil
}
