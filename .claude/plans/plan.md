# wellnown-env

**Single Source of Truth** for this project.

---

## Design Requirements

### Core Principle
**Every binary is a NATS node.** Dev laptop runs standalone, production connects as leaf to the cluster. Same code path.

### Stack
- **ardanlabs/conf** (v3) - Struct-based config with env vars + CLI flags
- **helmfile/vals** - Secret resolution from 25+ backends (Vault, AWS, 1Password, etc.)
- **NATS JetStream Embedded** - Every service embeds NATS (standalone or leaf node)

### Key Constraints
1. **GOWORK=off** - No go.work files
2. **No mocking** - Real testing with `ref+echo://` and `ref+file://` (vals built-ins)
3. **Runner independent** - Same binary works with Docker, Process Compose, k8s, systemd, bare metal
4. **GitHub namespace = Service namespace** - `org/repo` is the service identity
5. **Zero duplication** - Config struct IS the NATS registration schema

### Architecture
```
DEV LAPTOP                         PRODUCTION

┌─────────┐                       ┌─────────────────┐
│ Service │                       │   NATS Cluster  │
│ NATS    │  (standalone)         │   (hub)         │
│ node    │                       └────────┬────────┘
└─────────┘                                │
                                   ┌───────┼───────┐
                                   ▼       ▼       ▼
                               ┌─────┐ ┌─────┐ ┌─────┐
                               │Svc A│ │Svc B│ │Svc C│
                               │leaf │ │leaf │ │leaf │
                               └─────┘ └─────┘ └─────┘

Same binary. Same code path. Environment determines topology.
- No NATS_HUB → standalone (dev)
- NATS_HUB set → leaf node (prod)
```

---

## Overview

Build a Go library that unifies config, secrets, and service mesh. The config struct IS the schema that registers to the mesh.

## Zero Duplication Principle

**The exact same struct that defines the service on startup IS the exact same data sent to NATS.**

```go
type Config struct {
    conf.Version
    Server struct {
        Host string `conf:"default:0.0.0.0:8090"`
    }
    DB struct {
        Password string `conf:"mask,required"`
    }
}

// This struct is:
// 1. Parsed by ardanlabs/conf for env vars + CLI flags
// 2. Sent to NATS KV as-is (reflected to JSON)
// 3. The single source of truth for what this service needs
```

**No separate manifest. No Kubernetes ConfigMap schema. No documentation to maintain.**

The NATS server knows:
- What env vars every service needs (from conf tags)
- What secrets are required (from `mask` tag)
- What version is running (from ldflags)
- What dependencies exist (from `service:` tag)

All from ONE struct definition.

**Key capability**: The NATS registry becomes a **live documentation system**:
- Query what env vars any service needs (even if not running)
- Track env evolution across deployments
- Reach back to GitHub to see what a service version requires

## Change Detection (Dev + CI/CD)

Developers and GitHub workflows can detect when env/secret requirements change - **for their own service AND services they depend on**.

### Local Dev Check

```bash
# Check what changed in my service
wellnown-check --self

# Output:
# Your service (joeblew999/api-gateway) changes:
#   + APP_NEW_FEATURE (new required env)
#   - APP_OLD_FLAG (removed)
#   ~ APP_TIMEOUT (default: 5s → 10s)
#
# Services you depend on that changed:
#   ! joeblew999/auth-service added: AUTH_NEW_SCOPE (required)
#   ! joeblew999/billing-api removed: BILLING_OLD_KEY
```

### GitHub Workflow Check

```yaml
# .github/workflows/config-check.yml
name: Config Compatibility Check
on: [pull_request]

jobs:
  config-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build and extract schema
        run: |
          go build -o ./bin/service ./cmd/service
          ./bin/service --schema-dump > pr-schema.json

      - name: Check against live fleet
        env:
          NATS_URL: ${{ secrets.NATS_URL }}
        run: |
          wellnown-check \
            --repo ${{ github.repository }} \
            --pr-schema pr-schema.json \
            --check-deps \
            --check-consumers
```

### What Gets Checked

| Check | What it detects |
|-------|-----------------|
| **--self** | Changes in YOUR service's env/secret requirements |
| **--check-deps** | Changes in services YOU depend on |
| **--check-consumers** | Impact on services that depend on YOU |

### Example CI Output

```
Checking joeblew999/api-gateway PR #42...

YOUR SERVICE CHANGES:
  + APP_NEW_FEATURE (new required env)
  - APP_OLD_FLAG (removed)
  ~ APP_TIMEOUT (default: 5s → 10s)

DEPENDENCIES CHANGED (services you use):
  ⚠ joeblew999/auth-service
    + AUTH_NEW_SCOPE (required) - you may need to configure this

CONSUMERS AFFECTED (services that use you):
  ⚠ joeblew999/web-frontend (uses api-gateway)
  ⚠ joeblew999/mobile-backend (uses api-gateway)
  These services may need updates when you deploy.

❌ Breaking changes detected. Please review.
```

**Key insight**: Because NATS knows everyone's config structs, you can detect breaking changes BEFORE they hit production.

### Configuration Changelog - For Free

You get a configuration changelog without any extra work:

| Source | What it shows |
|--------|---------------|
| **Git** | Struct changes over time (what SHOULD be running) |
| **NATS** | Actual registrations (what IS running) |
| **Diff** | Gap between intent and reality |

```bash
# What changed in config between v1.0 and v1.1?
git diff v1.0..v1.1 -- config.go

# What's actually running in production?
nats kv get services_registry joeblew999.api-gateway.instance-1 | jq '.fields'

# Are they in sync?
wellnown-check --repo joeblew999/api-gateway --compare-live
```

**Three sources of truth that should match:**
1. `config.go` in git → defines what the service expects
2. NATS registry → shows what's actually running
3. Environment → provides the actual values

If they don't match, you have drift. The tooling catches it.

### NATS Bridges GitHub ↔ Running Instances

NATS knows BOTH:
1. **GitHub identity** (org, repo, commit, tag, branch) - embedded via ldflags at build time
2. **Runtime state** (host, port, instance ID, what's actually configured)

This means you can:

```bash
# "What env vars does the auth-service need?"
# → NATS tells you, because running instances registered their struct

# "What version of api-gateway is running in prod?"
# → NATS tells you: joeblew999/api-gateway @ v1.2.3 (commit abc123)

# "Did the env requirements change between what's running and what's in main?"
# → Compare NATS registration vs GitHub main branch
```

**The env that each service needs can evolve at dev time:**

| Time | What changes | Where it's recorded |
|------|--------------|---------------------|
| Dev time | Add `APP_NEW_VAR` to config struct | Git commit |
| Build time | ldflags embed GitHub identity | Binary |
| Runtime | Service registers struct to NATS | NATS KV |
| Query time | Compare any of the above | `wellnown-check` |

**Reaching back to GitHub from NATS:**

```go
// NATS registration includes GitHub identity
type ServiceRegistration struct {
    GitHub struct {
        Org    string `json:"org"`     // "joeblew999"
        Repo   string `json:"repo"`    // "api-gateway"
        Commit string `json:"commit"`  // "abc123def"
        Tag    string `json:"tag"`     // "v1.2.3"
        Branch string `json:"branch"`  // "main"
    }
    // ... fields from config struct
}

// You can look up the GitHub source for any running service
// NATS knows: "joeblew999/api-gateway @ v1.2.3 needs APP_DB_PASSWORD"
// GitHub shows: the actual config.go that defined that requirement
```

The NATS server becomes the **central knowledge graph** linking:
- What's in GitHub (code)
- What's in the binary (build)
- What's running (runtime)
- What's configured (environment)

## Build Configuration

```bash
# Disable go.work files
export GOWORK=off
```

## Runner Independence

The system is **completely runner-agnostic**. The binary doesn't know or care what started it:

| Runner | How it works |
|--------|--------------|
| **Docker/Compose** | Pass env vars via `environment:` |
| **Process Compose** | Pass env vars via `environment:` |
| **Kubernetes** | Pass env vars via ConfigMap/Secret |
| **systemd** | Pass env vars via `Environment=` |
| **Bare metal** | Export env vars or use `.env` file |
| **CI/CD** | GitHub Actions secrets, etc. |

Same binary, same config struct, different env vars per environment.

```yaml
# docker-compose.yml
services:
  api:
    image: myapp:latest
    environment:
      - APP_DB_PASSWORD=ref+vault://prod/db#password

# process-compose.yml
processes:
  api:
    command: ./myapp
    environment:
      APP_DB_PASSWORD: ref+1password://Dev/DB#password
```

## GitHub Namespace = Service Namespace

**Dead simple**: GitHub `org/repo` IS the service identity. No extra config needed.

```
# NATS KV keys follow GitHub namespace
services_registry/
├── joeblew999.api-gateway.instance-1
├── joeblew999.api-gateway.instance-2
├── joeblew999.auth-service.instance-1
├── joeblew999.worker.instance-1
└── somecorp.shared-lib.instance-1
```

### What You Can Query

**At runtime** (live services):
```bash
# What services are running?
nats kv ls services_registry

# What env vars does api-gateway need?
nats kv get services_registry joeblew999.api-gateway.instance-1 | jq '.fields'

# Who depends on auth-service?
nats kv ls services_registry | xargs -I{} sh -c \
  'nats kv get services_registry {} | jq -e ".fields[] | select(.dependency==\"joeblew999/auth-service\")" && echo {}'
```

**At dev time** (even if service isn't running):
```bash
# What did api-gateway v1.2.0 need? (historical from registry)
wellnown-check --repo joeblew999/api-gateway --tag v1.2.0 --show-env

# What will this PR change in terms of required env vars?
wellnown-check --repo joeblew999/api-gateway --pr-schema pr-schema.json
```

### Dependency Graph

Config struct declares dependencies explicitly:
```go
type Config struct {
    Dependencies struct {
        AuthService    string `conf:"service:joeblew999/auth-service"`
        BillingService string `conf:"service:joeblew999/billing-api"`
    }
}
```

This creates a queryable dependency graph in NATS - you know **who uses what**.

### Real-Time Service Updates

When a service you depend on changes (new version, new endpoint, config change), you get notified automatically:

```go
// Watch for auth-service updates in real-time
mgr.WatchService("joeblew999/auth-service", func(reg ServiceRegistration) {
    log.Printf("auth-service updated: %s at %s", reg.GitHub.Tag, reg.Instance.Host)

    // Update your internal client with new endpoint
    authClient.UpdateEndpoint(reg.Instance.Host)
})
```

**Use cases**:
- Service A discovers Service B's endpoint changed → auto-reconnect
- Service B deployed new version → Service A sees new env requirements
- Service C scaled up → load balancer sees new instances immediately
- Service D config changed → dependent services can react

No polling. No stale DNS. No manual updates. **NATS KV watch = push-based discovery**.

---

## Architecture: Embedded NATS in Every Service

**Every service embeds NATS** - there's no separate client vs server. Each wellnown-env instance IS a NATS node.

```
┌─────────────────────────────────────────────────────┐
│              NATS Hub (optional, for prod)          │
│         (central cluster for routing/persistence)   │
└─────────────────────┬───────────────────────────────┘
                      │ leaf connections
        ┌─────────────┼─────────────┐
        │             │             │
        ▼             ▼             ▼
   ┌─────────┐   ┌─────────┐   ┌─────────┐
   │ Svc A   │   │ Svc B   │   │ Svc C   │
   │ EMBEDDED│   │ EMBEDDED│   │ EMBEDDED│
   │ NATS    │   │ NATS    │   │ NATS    │
   └─────────┘   └─────────┘   └─────────┘
```

**Modes:**
| Mode | Use Case | Config |
|------|----------|--------|
| **Standalone** | Dev, testing, single service | `WithStandalone()` (default) |
| **Leaf Node** | Production, connects to hub | `WithHub("nats://hub:4222")` |

**KV replication** happens automatically across the leaf topology - all services see all registrations.

## Package Structure

```
wellnown-env/
├── manager.go          # Manager type, New(), Close(), embedded NATS
├── nats.go             # Embedded NATS server/leaf setup
├── parse.go            # Parse() - vals resolve + conf parse + register
├── register.go         # NATS KV registration
├── fields.go           # Struct reflection for field extraction
├── discovery.go        # WatchService, GetService, GetEnvRequirements
├── rotation.go         # Secret rotation subscription
├── options.go          # Functional options
├── errors.go           # Custom errors
├── doc.go              # Package docs
│
├── registry/
│   ├── types.go        # ServiceRegistration, FieldInfo
│   └── github.go       # GitOrg, GitRepo ldflags vars
│
├── cmd/
│   └── wellnown-check/
│       └── main.go     # CI config validation tool
│
└── examples/
    ├── basic/
    └── with-discovery/
```

---

## Implementation Phases

### Phase 1: Embedded NATS + conf + vals

**Files**: `manager.go`, `nats.go`, `parse.go`, `options.go`, `errors.go`

1. Create `Manager` struct that embeds NATS server (standalone or leaf)
2. Implement `resolveEnvSecrets()` - scan env vars for `ref+` prefixes, resolve via vals
3. Wrap `conf.Parse()` to call resolution first
4. Auto-create `services_registry` KV bucket on startup
5. Add functional options: `WithStandalone()`, `WithHub(url)`, `WithDataDir(path)`

**Key code** (`nats.go`):
```go
func (m *Manager) startNATS() error {
    opts := &server.Options{
        ServerName: m.instanceID,
        JetStream:  true,
        StoreDir:   m.opts.DataDir, // "" for in-memory
    }

    // Environment determines topology
    // NATS_HUB not set → standalone
    // NATS_HUB set → leaf node
    if hubURL := os.Getenv("NATS_HUB"); hubURL != "" {
        u, _ := url.Parse(hubURL)
        opts.LeafNode = server.LeafNodeOpts{
            Remotes: []*server.RemoteLeafOpts{{URLs: []*url.URL{u}}},
        }
    }

    m.server, err = server.NewServer(opts)
    m.server.Start()

    // Connect client to our own embedded server
    m.nc, err = nats.Connect(m.server.ClientURL())
    m.js, err = jetstream.New(m.nc)

    // Create KV bucket
    m.kv, err = m.js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
        Bucket: "services_registry",
    })
}
```

**Key code** (`parse.go`):
```go
func (m *Manager) Parse(cfg interface{}) (string, error) {
    // 1. Resolve ref+ env vars
    if err := m.resolveEnvSecrets(); err != nil {
        return "", fmt.Errorf("resolving secrets: %w", err)
    }
    // 2. Parse with ardanlabs/conf
    help, err := conf.Parse(m.prefix, cfg)
    // 3. Register to NATS KV
    if err == nil {
        m.register(cfg)
    }
    return help, err
}
```

### Phase 2: Registration + Field Extraction

**Files**: `register.go`, `fields.go`, `registry/types.go`, `registry/github.go`

1. Define `ServiceRegistration` type with GitHub identity, instance info, fields
2. Add ldflags variables: `GitOrg`, `GitRepo`, `GitCommit`, `GitTag`, `GitBranch`
3. Implement `extractFields()` using reflection to parse `conf` tags
4. Support custom `service:org/repo` tag for dependencies
5. Start heartbeat goroutine to refresh TTL

**Registry key format**: `{org}.{repo}.{instance-id}`

### Phase 3: Service Discovery

**Files**: `discovery.go`

1. `WatchService(name, callback)` - NATS KV watcher on `org.repo.*` pattern
2. `GetService(name)` - list all instances of a service
3. `GetAllServices()` - list all registered services
4. Return `Watcher` interface with `Stop()` method

### Phase 4: Secret Rotation

**Files**: `rotation.go`

1. `OnRotate(callback)` - subscribe to `secrets.rotated.>` NATS subject
2. `PublishRotation(path)` - for rotation service to notify
3. Typical handler: trigger graceful restart

### Phase 5: CI/CD Tooling

**Files**: `cmd/wellnown-check/main.go`, `schema.go`

1. `--dump-schema` - output config struct as JSON
2. `--pr-schema` - compare PR schema against live fleet
3. `--check-deps` - verify dependencies exist
4. `--check-consumers` - find who uses this service

### Phase 6: Rotation Service (Optional)

**Files**: `cmd/wellnown-rotation/main.go`

A standalone service that watches secret stores and publishes rotation events:

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

1. Poll secret refs periodically
2. Compare current vs last known value
3. Publish `secrets.rotated.<path>` when changed
4. Services subscribe and trigger graceful restart

### Phase 7: Binary Self-Update (Optional)

**Files**: `update/watcher.go`, GitHub Action for release publishing

Watch GitHub releases via NATS and auto-update:

```go
type UpdateWatcher struct {
    js      jetstream.JetStream
    org     string
    repo    string
    current string
}

func (w *UpdateWatcher) Watch() {
    // Subscribe to releases.{org}.{repo}
    // Compare semver, download new binary, restart
}
```

**GitHub Action** publishes to NATS on release:
```yaml
- name: Notify NATS
  run: |
    nats pub releases.${{ github.repository_owner }}.${{ github.event.repository.name }} \
      '{"tag": "${{ github.event.release.tag_name }}"}'
```

---

## Key Types

```go
// Manager coordinates config, secrets, and registration
type Manager struct {
    runtime    *vals.Runtime
    nc         *nats.Conn
    js         jetstream.JetStream
    kv         jetstream.KeyValue
    prefix     string
    instanceID string
    opts       Options
}

// ServiceRegistration is stored in NATS KV
type ServiceRegistration struct {
    GitHub   GitHubIdentity `json:"github"`
    Instance InstanceInfo   `json:"instance"`
    Fields   []FieldInfo    `json:"fields"`
}

// FieldInfo describes a config field
type FieldInfo struct {
    Path       string `json:"path"`       // "DB.Password"
    Type       string `json:"type"`       // "string"
    Default    string `json:"default"`    // "localhost"
    Required   bool   `json:"required"`
    IsSecret   bool   `json:"is_secret"`  // has mask tag
    EnvKey     string `json:"env_key"`    // "APP_DB_PASSWORD"
    Dependency string `json:"dependency"` // "org/repo" from service: tag
}
```

---

## Public API

```go
// Construction
func New(prefix string, opts ...Option) (*Manager, error)

// Config parsing (main entry point)
func (m *Manager) Parse(cfg interface{}) (string, error)

// Service discovery
func (m *Manager) WatchService(name string, fn func(ServiceRegistration)) (Watcher, error)
func (m *Manager) GetService(name string) ([]ServiceRegistration, error)

// Secret rotation
func (m *Manager) OnRotate(fn func()) error

// Cleanup
func (m *Manager) Close() error
```

---

## Dependencies

```go
require (
    github.com/ardanlabs/conf/v3 v3.1.7
    github.com/helmfile/vals v0.37.0
    github.com/nats-io/nats.go v1.31.0
    github.com/nats-io/nats-server/v2 v2.10.5  // Embedded server
    github.com/google/uuid v1.4.0
)
```

---

## Files to Create (in order)

### Phase 1: Foundation + Embedded NATS
1. `go.mod` - module definition and dependencies
2. `registry/types.go` - ServiceRegistration, FieldInfo, GitHubIdentity
3. `registry/github.go` - ldflags variables
4. `options.go` - Manager Options struct and functional options
5. `errors.go` - ErrHelpWanted, custom errors
6. `nats.go` - Embedded NATS server (standalone or leaf node)
7. `manager.go` - Manager struct, New(), Close()
8. `parse.go` - Parse(), resolveEnvSecrets()

### Phase 2-4: Core Features
9. `fields.go` - extractFields() reflection logic
10. `register.go` - register(), deregister(), heartbeat
11. `discovery.go` - WatchService(), GetService()
12. `rotation.go` - OnRotate()
13. `doc.go` - Package documentation

### Phase 5+: Tooling (Optional)
14. `cmd/wellnown-check/main.go` - CI config validation tool

### Examples
15. `examples/basic/main.go` - Basic usage (standalone mode)
16. `examples/with-hub/main.go` - Leaf node connecting to hub

---

## Example Usage

```go
// Same code everywhere - environment determines topology
mgr, _ := wellnown.New("APP") // reads NATS_HUB from env
defer mgr.Close()

var cfg Config
help, err := mgr.Parse(&cfg)
if errors.Is(err, wellnown.ErrHelpWanted) {
    fmt.Println(help)
    return
}

mgr.OnRotate(func() {
    syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
})
```

```bash
# Dev - standalone NATS node (in-memory)
./myservice

# Prod - leaf node connecting to hub
NATS_HUB=nats://cluster.example.com:4222 ./myservice
```

**Environment files**:
```bash
# Dev
APP_DB_PASSWORD=devpassword

# Prod
APP_DB_PASSWORD=ref+vault://prod/db#password
```

---

## Architecture Diagram

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
│                          │  (Embedded)     │                                │
│                          │                 │                                │
│                          │ • services.*    │  ← Service registrations       │
│                          │ • secrets.*     │  ← Rotation events             │
│                          │ • releases.*    │  ← Binary update notifications │
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

## Testing Strategy: Zero-Setup with vals Built-ins

**No mocks. No external services. vals has built-in providers for testing.**

### Built-in Test Providers

| Provider | Syntax | Use Case |
|----------|--------|----------|
| `ref+echo://` | `ref+echo://my-secret` | Returns the path as the value |
| `ref+file://` | `ref+file:///path/to/secret.txt` | Reads from local file |
| `ref+envsubst://` | `ref+envsubst://$OTHER_VAR` | Substitutes env vars |

### Test Examples

```bash
# echo - returns "supersecret" as the resolved value
APP_DB_PASSWORD=ref+echo://supersecret

# file - reads from testdata/secrets/db_password.txt
APP_DB_PASSWORD=ref+file://./testdata/secrets/db_password.txt

# Run tests - no setup required!
go test ./...
```

### Test Config Example

```go
func TestSecretResolution(t *testing.T) {
    // Use echo provider - no external dependencies
    t.Setenv("APP_DB_PASSWORD", "ref+echo://test-db-password")
    t.Setenv("APP_API_KEY", "ref+echo://test-api-key")

    mgr, err := wellnown.New("APP")
    require.NoError(t, err)

    var cfg Config
    _, err = mgr.Parse(&cfg)
    require.NoError(t, err)

    assert.Equal(t, "test-db-password", cfg.DB.Password)
    assert.Equal(t, "test-api-key", cfg.API.Key)
}
```

### File-Based Secrets for Complex Tests

```
testdata/
└── secrets/
    ├── db_password.txt      # Contains: supersecret
    ├── api_key.txt          # Contains: abc123
    └── config.json          # For JSON fragment extraction
```

```go
func TestFileSecrets(t *testing.T) {
    // Read from test files
    t.Setenv("APP_DB_PASSWORD", "ref+file://./testdata/secrets/db_password.txt")

    mgr, err := wellnown.New("APP")
    require.NoError(t, err)

    var cfg Config
    _, err = mgr.Parse(&cfg)
    require.NoError(t, err)

    assert.Equal(t, "supersecret", cfg.DB.Password)
}
```

### CI Workflow (plain bash)

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'

      - name: Test
        run: go test -v ./...
```

**That's it.** No Vault, no 1Password, no AWS. Same tests run locally and in CI with zero setup.

### Production Testing (Optional)

For integration tests against real secret stores:

```bash
# Vault (if you need it)
vault server -dev -dev-root-token-id="test-token" &
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=test-token
APP_DB_PASSWORD=ref+vault://secret/data/test#db_password go test -v ./... -tags=integration

# 1Password (if you have it)
APP_DB_PASSWORD=ref+1password://Dev/Database#password go test -v ./... -tags=integration
```

---

## Environment Variables Reference

### Core

| Variable | Description |
|----------|-------------|
| `NATS_URL` | NATS server URL (or embedded) |
| `APP_*` | All config fields (prefix configurable) |

### vals Providers (Production)

| Provider | Required Variables |
|----------|-----------|
| **1Password** | `OP_SERVICE_ACCOUNT_TOKEN` |
| **AWS** | `AWS_PROFILE`, `AWS_DEFAULT_REGION` |
| **GCP** | `GOOGLE_APPLICATION_CREDENTIALS`, `GCP_PROJECT` |
| **Azure** | `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET` |

### Build-time (ldflags)

| Variable | Description |
|----------|-------------|
| `GitCommit` | Git commit hash |
| `GitTag` | Git tag/version |
| `GitBranch` | Git branch |
| `GitOrg` | GitHub organization |
| `GitRepo` | GitHub repository name |
