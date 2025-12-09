# Unified Config, Secrets & Service Mesh System for Go

## Overview

A complete system that unifies configuration management, secrets resolution, and service discovery into a single pattern. The core insight: **the same struct that defines your service's config IS the schema that gets registered to the service mesh**.

No duplication. No drift. Dev and production use identical binaries with different environment variables.

---

## Part 1: The Problem

### The Tension Between CLI and Server Config

Traditionally there's friction between:
- CLI flag handling
- Environment variable loading
- Default values
- Secrets management
- Documentation of what config a service needs

These are often handled by different systems, leading to:
- Duplicate definitions
- Config drift between environments
- Runtime failures when secrets are missing
- No visibility into what services need what

---

## Part 2: ardanlabs/conf — The Foundation

GitHub: https://github.com/ardanlabs/conf

### What It Does

Parses a single struct for both environment variables and command-line flags.

### Struct Tags

```go
default  - Provides the default value for the help
env      - Allows for overriding the default variable name
flag     - Allows for overriding the default flag name
short    - Denotes a shorthand option for the flag
noprint  - Denotes to not include the field in any display string
mask     - Includes the field in any display string but masks out the value
required - Denotes an overriding value must be provided
notzero  - Denotes a field can't be set to its zero value
help     - Provides a description for the help
```

### Basic Usage

```go
type Config struct {
    conf.Version
    Server struct {
        Host            string        `conf:"default:0.0.0.0:8090"`
        ShutdownTimeout time.Duration `conf:"default:20s"`
    }
    NATS struct {
        URL   string `conf:"default:nats://localhost:4222"`
        Creds string `conf:"mask,required"`
    }
    DB struct {
        Host     string `conf:"default:localhost:5432"`
        Password string `conf:"mask,required"`
    }
    Debug bool `conf:"default:false,short:d"`
}

func main() {
    cfg := Config{
        Version: conf.Version{Build: "v1.0.0", Desc: "My Service"},
    }
    
    help, err := conf.Parse("APP", &cfg)
    if err != nil {
        if errors.Is(err, conf.ErrHelpWanted) {
            fmt.Println(help)
            os.Exit(0)
        }
        log.Fatalf("config: %v", err)
    }
    
    // Use cfg.Server.Host, cfg.NATS.URL, etc.
}
```

### Three Ways to Set Config (Same Binary)

```bash
# 1. Environment variables (12-factor, good for containers)
export APP_SERVER_HOST=0.0.0.0:9000
export APP_NATS_URL=nats://prod:4222
./myservice

# 2. CLI flags (good for dev/testing)
./myservice --server-host=0.0.0.0:9000 --nats-url=nats://prod:4222

# 3. Mix both (CLI overrides env)
export APP_NATS_URL=nats://prod:4222
./myservice --debug
```

**Precedence:** CLI flags → Environment variables → Defaults

### Auto-Generated Help

```
Usage: myservice [options...] [arguments...]

OPTIONS
      --server-host              <string>    (default: 0.0.0.0:8090)
      --server-shutdown-timeout  <duration>  (default: 20s)
      --nats-url                 <string>    (default: nats://localhost:4222)
      --nats-creds               <string>    (required)
      --db-host                  <string>    (default: localhost:5432)
      --db-password              <string>    (required)
  -d, --debug                    <bool>      (default: false)
  -h, --help                                 display this help message
  -v, --version                              display version

ENVIRONMENT
  APP_SERVER_HOST
  APP_SERVER_SHUTDOWN_TIMEOUT
  APP_NATS_URL
  APP_NATS_CREDS
  APP_DB_HOST
  APP_DB_PASSWORD
  APP_DEBUG
```

---

## Part 3: helmfile/vals — Secret Resolution

GitHub: https://github.com/helmfile/vals

### What It Does

Resolves `ref+<provider>://path` URIs to actual secret values from any backend.

### Supported Backends

- Vault
- AWS SSM Parameter Store
- AWS Secrets Manager
- AWS S3 / KMS
- GCP Secrets Manager / KMS
- 1Password / 1Password Connect
- Azure Key Vault
- Doppler
- SOPS-encrypted files
- Terraform State
- Kubernetes Secrets
- Conjur
- HCP Vault Secrets
- Bitwarden
- And many more...

### Expression Syntax

```
ref+BACKEND://PATH[?PARAMS][#FRAGMENT]
```

Examples:
```bash
ref+vault://secret/data/myapp#/db_password
ref+awssecrets://myapp/prod#/api_key
ref+1password://Private/MyApp#password
ref+awsssm://myapp/config?region=us-west-2#/database/host
ref+gcpsecrets://myproject/mysecret?version=3
```

### Go API

```go
import "github.com/helmfile/vals"

runtime, err := vals.New(vals.Options{CacheSize: 256})
if err != nil {
    return err
}

// Resolve a single reference
secret, err := runtime.Get("ref+vault://secret/data/myapp#/password")

// Or evaluate a whole map
result, err := runtime.Eval(map[string]interface{}{
    "db_password": "ref+vault://secret/data/myapp#/db_password",
    "api_key":     "ref+1password://Private/MyApp#api_key",
})
```

---

## Part 4: The Unified System — Combining conf + vals + NATS

### The Core Idea

1. **conf** defines the struct with tags
2. **vals** resolves any `ref+` secret references in env vars
3. **NATS** receives the registration of what this service needs
4. Same struct serves all three purposes — no duplication

### The Config Manager

```go
// pkg/config/config.go
package config

import (
    "encoding/json"
    "fmt"
    "os"
    "reflect"
    "strings"
    "time"
    
    "github.com/ardanlabs/conf/v3"
    "github.com/helmfile/vals"
    "github.com/nats-io/nats.go"
)

type Manager struct {
    runtime   *vals.Runtime
    nc        *nats.Conn
    js        nats.JetStreamContext
    kv        nats.KeyValue
    prefix    string
    service   string
    onRotate  func()
}

func New(natsURL, prefix, service string) (*Manager, error) {
    // vals runtime for secret resolution
    runtime, err := vals.New(vals.Options{CacheSize: 256})
    if err != nil {
        return nil, err
    }
    
    // NATS connection for registration and rotation notifications
    nc, err := nats.Connect(natsURL)
    if err != nil {
        return nil, err
    }
    
    js, err := nc.JetStream()
    if err != nil {
        return nil, err
    }
    
    // KV store for service registry
    kv, err := js.KeyValue("services_registry")
    if err != nil {
        // Create if doesn't exist
        kv, err = js.CreateKeyValue(&nats.KeyValueConfig{
            Bucket: "services_registry",
            TTL:    time.Hour, // Services must heartbeat
        })
        if err != nil {
            return nil, err
        }
    }
    
    return &Manager{
        runtime: runtime,
        nc:      nc,
        js:      js,
        kv:      kv,
        prefix:  prefix,
        service: service,
    }, nil
}

// Parse wraps ardanlabs/conf but resolves vals refs first and registers
func (m *Manager) Parse(cfg interface{}) (string, error) {
    // 1. Resolve any ref+ env vars using vals
    if err := m.resolveEnvRefs(); err != nil {
        return "", fmt.Errorf("resolving secrets: %w", err)
    }
    
    // 2. Parse with ardanlabs/conf (now sees resolved values)
    help, err := conf.Parse(m.prefix, cfg)
    if err != nil {
        return help, err
    }
    
    // 3. Register to NATS — same struct, reflected
    if err := m.register(cfg); err != nil {
        return help, fmt.Errorf("registering service: %w", err)
    }
    
    // 4. Start heartbeat
    go m.heartbeat(cfg)
    
    return help, nil
}

func (m *Manager) resolveEnvRefs() error {
    for _, env := range os.Environ() {
        pair := strings.SplitN(env, "=", 2)
        if len(pair) != 2 {
            continue
        }
        key, value := pair[0], pair[1]
        
        if strings.HasPrefix(value, "ref+") {
            resolved, err := m.runtime.Get(value)
            if err != nil {
                return fmt.Errorf("resolving %s: %w", key, err)
            }
            os.Setenv(key, resolved)
        }
    }
    return nil
}

// Subscribe to rotation notifications
func (m *Manager) OnRotate(fn func()) error {
    m.onRotate = fn
    _, err := m.nc.Subscribe("secrets.rotated.>", func(msg *nats.Msg) {
        if m.onRotate != nil {
            m.onRotate()
        }
    })
    return err
}

func (m *Manager) Close() error {
    // Deregister
    m.kv.Delete(m.registryKey())
    return m.nc.Close()
}
```

### Service Registration

```go
type ServiceRegistration struct {
    // Identity — where did I come from?
    GitHub struct {
        Org    string `json:"org"`
        Repo   string `json:"repo"`
        Commit string `json:"commit"`
        Tag    string `json:"tag"`
        Branch string `json:"branch"`
    } `json:"github"`
    
    // Runtime — where am I now?
    Instance struct {
        ID        string    `json:"id"`
        Host      string    `json:"host"`
        StartedAt time.Time `json:"started_at"`
    } `json:"instance"`
    
    // Config — what do I need?
    Fields []FieldInfo `json:"fields"`
}

type FieldInfo struct {
    Path       string `json:"path"`        // "DB.Password"
    Type       string `json:"type"`        // "string", "duration"
    Default    string `json:"default"`     // "localhost:5432"
    Required   bool   `json:"required"`
    IsSecret   bool   `json:"is_secret"`   // has mask tag
    Source     string `json:"source"`      // "env", "flag", "default"
    EnvKey     string `json:"env_key"`     // "APP_DB_PASSWORD"
    Dependency string `json:"dependency"`  // "gedw99/auth-service" if service tag
}

func (m *Manager) register(cfg interface{}) error {
    reg := ServiceRegistration{
        Instance: struct {
            ID        string    `json:"id"`
            Host      string    `json:"host"`
            StartedAt time.Time `json:"started_at"`
        }{
            ID:        uuid.New().String(),
            Host:      getHostPort(),
            StartedAt: time.Now(),
        },
    }
    
    // Set from build-time ldflags
    reg.GitHub.Org = GitOrg
    reg.GitHub.Repo = GitRepo
    reg.GitHub.Commit = GitCommit
    reg.GitHub.Tag = GitTag
    reg.GitHub.Branch = GitBranch
    
    // Reflect over config struct to extract field info
    reg.Fields = m.extractFields(cfg)
    
    data, err := json.Marshal(reg)
    if err != nil {
        return err
    }
    
    _, err = m.kv.Put(m.registryKey(), data)
    return err
}

func (m *Manager) registryKey() string {
    return fmt.Sprintf("%s.%s.%s", GitOrg, GitRepo, m.instanceID)
}

func (m *Manager) extractFields(cfg interface{}) []FieldInfo {
    var fields []FieldInfo
    
    val := reflect.ValueOf(cfg)
    if val.Kind() == reflect.Ptr {
        val = val.Elem()
    }
    
    m.walkStruct(val, "", &fields)
    return fields
}

func (m *Manager) walkStruct(val reflect.Value, prefix string, fields *[]FieldInfo) {
    typ := val.Type()
    
    for i := 0; i < val.NumField(); i++ {
        field := typ.Field(i)
        fieldVal := val.Field(i)
        
        path := field.Name
        if prefix != "" {
            path = prefix + "." + field.Name
        }
        
        // Recurse into nested structs
        if field.Type.Kind() == reflect.Struct && field.Type.Name() != "Duration" {
            m.walkStruct(fieldVal, path, fields)
            continue
        }
        
        // Parse conf tag
        confTag := field.Tag.Get("conf")
        info := FieldInfo{
            Path:   path,
            Type:   field.Type.String(),
            EnvKey: m.prefix + "_" + strings.ToUpper(strings.ReplaceAll(path, ".", "_")),
        }
        
        // Parse tag parts
        for _, part := range strings.Split(confTag, ",") {
            part = strings.TrimSpace(part)
            if strings.HasPrefix(part, "default:") {
                info.Default = strings.TrimPrefix(part, "default:")
            } else if part == "required" {
                info.Required = true
            } else if part == "mask" {
                info.IsSecret = true
            } else if strings.HasPrefix(part, "service:") {
                info.Dependency = strings.TrimPrefix(part, "service:")
            }
        }
        
        *fields = append(*fields, info)
    }
}

func (m *Manager) heartbeat(cfg interface{}) {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        m.register(cfg) // Re-register to refresh TTL
    }
}
```

### Service Discovery via Watch

```go
func (m *Manager) WatchService(name string, fn func(reg ServiceRegistration)) error {
    watcher, err := m.kv.Watch(name + ".*")
    if err != nil {
        return err
    }
    
    go func() {
        for entry := range watcher.Updates() {
            if entry == nil {
                continue
            }
            var reg ServiceRegistration
            if err := json.Unmarshal(entry.Value(), &reg); err != nil {
                continue
            }
            fn(reg)
        }
    }()
    
    return nil
}

func (m *Manager) GetService(name string) ([]ServiceRegistration, error) {
    keys, err := m.kv.Keys()
    if err != nil {
        return nil, err
    }
    
    var regs []ServiceRegistration
    for _, key := range keys {
        if strings.HasPrefix(key, name+".") {
            entry, err := m.kv.Get(key)
            if err != nil {
                continue
            }
            var reg ServiceRegistration
            if err := json.Unmarshal(entry.Value(), &reg); err != nil {
                continue
            }
            regs = append(regs, reg)
        }
    }
    
    return regs, nil
}
```

---

## Part 5: Complete Usage Example

### Build Configuration (Makefile)

```makefile
VERSION := $(shell git describe --tags --always)
COMMIT  := $(shell git rev-parse HEAD)
BRANCH  := $(shell git rev-parse --abbrev-ref HEAD)
REPO    := $(shell git remote get-url origin | sed 's/.*github.com[:/]\(.*\)\.git/\1/')
ORG     := $(shell echo $(REPO) | cut -d'/' -f1)
NAME    := $(shell echo $(REPO) | cut -d'/' -f2)

LDFLAGS := -ldflags "\
    -X main.GitCommit=$(COMMIT) \
    -X main.GitTag=$(VERSION) \
    -X main.GitBranch=$(BRANCH) \
    -X main.GitOrg=$(ORG) \
    -X main.GitRepo=$(NAME)"

build:
	go build $(LDFLAGS) -o ./bin/$(NAME) ./cmd/$(NAME)
```

### Main Application

```go
package main

import (
    "errors"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/ardanlabs/conf/v3"
    "yourproject/pkg/config"
)

// Set via ldflags
var (
    GitCommit string
    GitTag    string
    GitBranch string
    GitOrg    string
    GitRepo   string
)

type Config struct {
    conf.Version
    Server struct {
        Host            string        `conf:"default:0.0.0.0:8090"`
        ShutdownTimeout time.Duration `conf:"default:20s"`
    }
    NATS struct {
        URL   string `conf:"default:nats://localhost:4222"`
        Creds string `conf:"mask"`
    }
    DB struct {
        Host     string `conf:"default:localhost:5432"`
        Password string `conf:"mask,required"`
    }
    Dependencies struct {
        AuthService    string `conf:"service:gedw99/auth-service"`
        BillingService string `conf:"service:gedw99/billing-api"`
    }
    Debug bool `conf:"default:false,short:d"`
}

func main() {
    if err := run(); err != nil {
        log.Fatal(err)
    }
}

func run() error {
    // Initialize config manager
    mgr, err := config.New(
        os.Getenv("NATS_URL"),
        "APP",
        GitRepo,
    )
    if err != nil {
        return fmt.Errorf("creating config manager: %w", err)
    }
    defer mgr.Close()
    
    // Parse config — resolves secrets, registers to NATS
    var cfg Config
    cfg.Version = conf.Version{
        Build: GitTag,
        Desc:  "API Gateway Service",
    }
    
    help, err := mgr.Parse(&cfg)
    if err != nil {
        if errors.Is(err, conf.ErrHelpWanted) {
            fmt.Println(help)
            return nil
        }
        return fmt.Errorf("parsing config: %w", err)
    }
    
    // Log config (secrets masked)
    log.Println("Starting with config:")
    log.Println(conf.String(&cfg))
    
    // Watch for secret rotation
    mgr.OnRotate(func() {
        log.Println("secrets rotated, initiating graceful shutdown")
        // Signal main to restart
        syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
    })
    
    // Watch for dependent services
    mgr.WatchService("gedw99/auth-service", func(reg config.ServiceRegistration) {
        log.Printf("auth-service updated: %s at %s", reg.GitHub.Tag, reg.Instance.Host)
        // Update internal service URL
    })
    
    // Run your service...
    log.Printf("Service running at %s", cfg.Server.Host)
    
    // Wait for shutdown signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Shutting down...")
    return nil
}
```

### Environment Files

**Development (.env.dev):**
```bash
NATS_URL=nats://localhost:4222
APP_DB_PASSWORD=devpassword
APP_DEBUG=true
```

**Staging (.env.staging):**
```bash
NATS_URL=nats://nats.staging:4222
APP_DB_PASSWORD=ref+1password://Staging/DB#password
APP_NATS_CREDS=ref+1password://Staging/NATS#creds
```

**Production (.env.prod):**
```bash
NATS_URL=nats://nats.prod:4222
APP_DB_PASSWORD=ref+vault://prod/db#password
APP_NATS_CREDS=ref+vault://prod/nats#creds
```

---

## Part 6: Runner Independence

The binary doesn't know or care what started it. Same binary works everywhere:

### Docker Compose

```yaml
version: '3.8'
services:
  api:
    image: gedw99/api-gateway:latest
    environment:
      - NATS_URL=nats://nats:4222
      - APP_DB_PASSWORD=ref+vault://prod/db#password
```

### Process Compose

```yaml
version: "0.5"
processes:
  api:
    command: ./api-gateway
    environment:
      NATS_URL: nats://localhost:4222
      APP_DB_PASSWORD: ref+1password://Dev/DB#password
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
        - name: api
          image: gedw99/api-gateway:latest
          env:
            - name: NATS_URL
              value: nats://nats.prod:4222
            - name: APP_DB_PASSWORD
              value: ref+vault://prod/db#password
```

### Systemd

```ini
[Unit]
Description=API Gateway
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/api-gateway
Environment=NATS_URL=nats://localhost:4222
Environment=APP_DB_PASSWORD=ref+vault://prod/db#password
Restart=always

[Install]
WantedBy=multi-user.target
```

---

## Part 7: GitHub Namespace = Service Namespace

The GitHub org/repo IS the service identity:

```
services.registry.gedw99.api-gateway.instance-1
services.registry.gedw99.api-gateway.instance-2
services.registry.gedw99.auth-service.instance-1
services.registry.gedw99.worker.instance-1
services.registry.somecorp.shared-lib.instance-1
```

### Querying the Registry

```bash
# What services exist?
nats kv ls services_registry

# What does api-gateway need?
nats kv get services_registry gedw99.api-gateway.instance-1

# What's using DB_PASSWORD?
nats kv ls services_registry | xargs -I{} sh -c \
  'nats kv get services_registry {} | grep -l DB_PASSWORD && echo {}'
```

### Dependency Declaration in Config

```go
type Config struct {
    conf.Version
    
    Dependencies struct {
        // References GitHub namespaces directly
        Auth    string `conf:"service:gedw99/auth-service"`
        Billing string `conf:"service:gedw99/billing-api"`
        Shared  string `conf:"service:somecorp/shared-lib"` // External
    }
}
```

---

## Part 8: CI/CD Integration — Checking Against Live Fleet

### GitHub Action

```yaml
name: Config Compatibility Check

on: [pull_request]

jobs:
  config-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Build and extract schema
        run: |
          go build -o ./bin/service ./cmd/service
          ./bin/service --schema-dump > pr-schema.json
      
      - name: Check against live fleet
        env:
          NATS_URL: ${{ secrets.NATS_URL }}
          NATS_CREDS: ${{ secrets.NATS_CREDS }}
        run: |
          go run ./cmd/config-check \
            --repo ${{ github.repository }} \
            --pr-schema pr-schema.json \
            --check-deps \
            --check-consumers
```

### The Config Check Tool

```go
// cmd/config-check/main.go
package main

func main() {
    repo := flag.String("repo", "", "GitHub repo (org/name)")
    prSchema := flag.String("pr-schema", "", "Path to PR schema JSON")
    checkDeps := flag.Bool("check-deps", false, "Check dependencies")
    checkConsumers := flag.Bool("check-consumers", false, "Check consumers")
    flag.Parse()
    
    // Connect to NATS
    nc, _ := nats.Connect(os.Getenv("NATS_URL"))
    js, _ := nc.JetStream()
    kv, _ := js.KeyValue("services_registry")
    
    // Load PR schema
    var pr Schema
    data, _ := os.ReadFile(*prSchema)
    json.Unmarshal(data, &pr)
    
    // Get live schema
    live := getLiveSchema(kv, *repo)
    
    // Check for breaking changes
    issues := checkBreakingChanges(live, pr, kv, *checkDeps, *checkConsumers)
    
    // Report
    printReport(issues)
    
    if hasBreaking(issues) {
        os.Exit(1)
    }
}

func checkBreakingChanges(live, pr Schema, kv nats.KeyValue, checkDeps, checkConsumers bool) []Issue {
    var issues []Issue
    
    // Check removed fields
    for _, field := range live.Fields {
        if !pr.HasField(field.Path) {
            issue := Issue{
                Severity: "warning",
                Message:  fmt.Sprintf("Field %s removed", field.EnvKey),
            }
            
            // Check if any consumer uses this
            if checkConsumers {
                consumers := findConsumers(kv, live.Repo)
                for _, c := range consumers {
                    if c.Uses(field.EnvKey) {
                        issue.Severity = "breaking"
                        issue.Message = fmt.Sprintf(
                            "Field %s removed but %s depends on it",
                            field.EnvKey, c.Repo,
                        )
                    }
                }
            }
            
            issues = append(issues, issue)
        }
    }
    
    // Check new required fields
    for _, field := range pr.Fields {
        if field.Required && !live.HasField(field.Path) {
            issues = append(issues, Issue{
                Severity: "warning",
                Message:  fmt.Sprintf(
                    "New required field %s — ensure configured before deploy",
                    field.EnvKey,
                ),
            })
        }
    }
    
    // Check dependencies exist
    if checkDeps {
        for _, field := range pr.Fields {
            if field.Dependency != "" {
                if !serviceExists(kv, field.Dependency) {
                    issues = append(issues, Issue{
                        Severity: "error",
                        Message:  fmt.Sprintf(
                            "Dependency %s not found in live fleet",
                            field.Dependency,
                        ),
                    })
                }
            }
        }
    }
    
    return issues
}
```

### Example CI Output

```
Checking gedw99/api-gateway PR #42...

This service changes:
  + APP_NEW_FEATURE (new required env)
  - APP_OLD_FLAG (removed)
  ~ APP_TIMEOUT (default: 5s → 10s)

Services that depend on gedw99/api-gateway:
  ⚠ gedw99/web-frontend (uses api-gateway, may need update)
  ⚠ gedw99/mobile-backend (uses api-gateway, may need update)

Services this PR depends on:
  ✓ gedw99/auth-service (running v1.2.0, compatible)
  ✗ gedw99/billing-api (PR requires APP_BILLING_V2, not in live registry)

❌ Breaking changes detected. Please review.
```

---

## Part 9: Secret Rotation Flow

### The Problem

vals resolves secrets at startup. If secrets rotate in Vault/1Password, running services have stale values.

### The Solution

A rotation service watches secret stores and notifies via NATS:

```
┌─────────────────┐     webhook/poll     ┌──────────────────┐
│  Secret Store   │ ──────────────────▶  │  Rotation Svc    │
│  (1Password,    │                      │  (vals + NATS)   │
│   Vault, etc)   │                      └────────┬─────────┘
└─────────────────┘                               │
                                                  │ NATS publish
                                                  │ "secrets.rotated.*"
                                                  ▼
                              ┌─────────────────────────────────────┐
                              │           NATS JetStream            │
                              └─────────────────────────────────────┘
                                    │           │           │
                                    ▼           ▼           ▼
                              ┌─────────┐ ┌─────────┐ ┌─────────┐
                              │ Svc A   │ │ Svc B   │ │ Svc C   │
                              │ restart │ │ restart │ │ restart │
                              └─────────┘ └─────────┘ └─────────┘
```

### Rotation Service

```go
// cmd/rotation-service/main.go
package main

type SecretWatcher struct {
    runtime  *vals.Runtime
    js       nats.JetStreamContext
    secrets  map[string]string // ref -> last known value
    interval time.Duration
}

func (w *SecretWatcher) Watch(refs []string) {
    ticker := time.NewTicker(w.interval)
    
    for range ticker.C {
        for _, ref := range refs {
            current, err := w.runtime.Get(ref)
            if err != nil {
                log.Printf("error resolving %s: %v", ref, err)
                continue
            }
            
            if last, ok := w.secrets[ref]; ok && last != current {
                log.Printf("secret rotated: %s", ref)
                
                // Publish rotation event
                w.js.Publish("secrets.rotated", []byte(ref))
                
                // Could also publish targeted:
                // w.js.Publish("secrets.rotated.db-password", []byte(ref))
            }
            
            w.secrets[ref] = current
        }
    }
}
```

### Service Handling Rotation

```go
mgr.OnRotate(func() {
    log.Println("secrets rotated, initiating graceful shutdown")
    syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
})
```

The orchestrator (systemd, k8s, docker) restarts the service, which re-resolves secrets on startup.

---

## Part 10: Binary Self-Update

### Combining with GitHub Release Watching

```go
type UpdateWatcher struct {
    js       nats.JetStreamContext
    org      string
    repo     string
    current  string
}

func (w *UpdateWatcher) Watch() {
    // Subscribe to release notifications
    w.js.Subscribe("releases."+w.org+"."+w.repo, func(msg *nats.Msg) {
        var release Release
        json.Unmarshal(msg.Data, &release)
        
        if semver.Compare(release.Tag, w.current) > 0 {
            log.Printf("new version available: %s (current: %s)", 
                release.Tag, w.current)
            
            // Download and replace binary
            if err := w.selfUpdate(release); err != nil {
                log.Printf("update failed: %v", err)
                return
            }
            
            // Restart
            syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
        }
    })
}
```

### Release Publisher (GitHub Action)

```yaml
name: Publish Release

on:
  release:
    types: [published]

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - name: Notify NATS
        env:
          NATS_URL: ${{ secrets.NATS_URL }}
        run: |
          nats pub releases.${{ github.repository_owner }}.${{ github.event.repository.name }} \
            '{"tag": "${{ github.event.release.tag_name }}", "url": "${{ github.event.release.html_url }}"}'
```

---

## Part 11: Full Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Control Plane                                   │
│                                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │    GitHub    │  │    Vault/    │  │   NATS KV    │  │   Rotation   │    │
│  │   Releases   │  │  1Password   │  │   Registry   │  │   Service    │    │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘    │
│         │                 │                 │                 │            │
│         └─────────────────┴─────────────────┴─────────────────┘            │
│                                    │                                        │
│                                    ▼                                        │
│                          ┌─────────────────┐                                │
│                          │  NATS JetStream │                                │
│                          │                 │                                │
│                          │ • services.*    │                                │
│                          │ • secrets.*     │                                │
│                          │ • releases.*    │                                │
│                          │ • config.*      │                                │
│                          └────────┬────────┘                                │
└───────────────────────────────────┼─────────────────────────────────────────┘
                                    │
            ┌───────────────────────┼───────────────────────┐
            │                       │                       │
            ▼                       ▼                       ▼
    ┌───────────────┐       ┌───────────────┐       ┌───────────────┐
    │   Service A   │       │   Service B   │       │   Service C   │
    │               │       │               │       │               │
    │ • Self-register│      │ • Self-register│      │ • Self-register│
    │ • Watch deps  │       │ • Watch deps  │       │ • Watch deps  │
    │ • Watch secrets│      │ • Watch secrets│      │ • Watch secrets│
    │ • Self-update │       │ • Self-update │       │ • Self-update │
    └───────────────┘       └───────────────┘       └───────────────┘
            │                       │                       │
    ┌───────┴───────┐       ┌───────┴───────┐       ┌───────┴───────┐
    │    Runner     │       │    Runner     │       │    Runner     │
    │   (Docker/    │       │   (k8s/       │       │  (systemd/    │
    │   compose)    │       │   pod)        │       │   bare)       │
    └───────────────┘       └───────────────┘       └───────────────┘
```

---

## Part 12: What You Get

### For Developers

| Before | After |
|--------|-------|
| Separate config handling per service | Import one package |
| Different patterns dev vs prod | Same binary everywhere |
| Manual documentation of env vars | Auto-generated from struct |
| "Works on my machine" | Identical config parsing |
| Missing secret = runtime crash | Missing secret = startup failure |
| No visibility into dependencies | Query NATS for full graph |

### For Operations

| Before | After |
|--------|-------|
| Coordinate secret rotation manually | NATS broadcast, automatic restart |
| SSH to check what's running | Query NATS registry |
| Guess at service dependencies | Explicit in registration |
| Config drift between environments | Single source of truth |
| Manual version tracking | Live in registry |

### For CI/CD

| Before | After |
|--------|-------|
| Breaking changes found in prod | Caught at PR time |
| Manual dependency tracking | Automatic from struct tags |
| Hope secrets are configured | Validate against live fleet |
| Separate manifest files | Config struct IS the manifest |

---

## Part 13: Summary

### Core Packages

1. **ardanlabs/conf** — Struct-based config with env + CLI + defaults
2. **helmfile/vals** — Secret resolution from any backend
3. **NATS JetStream** — Registration, discovery, rotation events

### Key Principles

1. **Single struct defines everything** — config schema, env vars, CLI flags, secrets, dependencies
2. **Same struct registers to NATS** — no separate manifest
3. **GitHub namespace = service namespace** — org/repo is the identity
4. **Runner independent** — binary doesn't know what started it
5. **Fail fast** — missing secrets caught at startup, not runtime
6. **Live fleet awareness** — CI can check against what's actually running

### The One-Liner

> The `conf` struct is the schema, vals resolves secrets, NATS distributes the state. Same code, dev to prod.

---

## Appendix: Environment Variables Reference

### Core

| Variable | Description |
|----------|-------------|
| `NATS_URL` | NATS server URL |
| `APP_*` | All config fields (prefix configurable) |

### vals Providers

| Provider | Variables |
|----------|-----------|
| Vault | `VAULT_ADDR`, `VAULT_TOKEN`, `VAULT_NAMESPACE` |
| 1Password | `OP_SERVICE_ACCOUNT_TOKEN` |
| AWS | `AWS_PROFILE`, `AWS_DEFAULT_REGION` |
| GCP | `GOOGLE_APPLICATION_CREDENTIALS`, `GCP_PROJECT` |

### Build-time (ldflags)

| Variable | Description |
|----------|-------------|
| `GitCommit` | Git commit hash |
| `GitTag` | Git tag/version |
| `GitBranch` | Git branch |
| `GitOrg` | GitHub organization |
| `GitRepo` | GitHub repository name |