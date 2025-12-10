# wellknown-env

**Single Source of Truth** for this project.

---

## Vision

A platform + SDK that gives developers:
1. **Config from a struct** - define what your service needs
2. **Secrets resolved automatically** - vals handles 25+ backends
3. **Service mesh for free** - register, discover, watch dependencies
4. **Ops GUI for free** - see your config, secrets, dependencies in real-time
5. **Runtime updates** - know when services you depend on change

**Self-hosted or cloud-hosted** - same code, same experience.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                     THE PLATFORM                                     │
│                                                                      │
│  ┌─────────────┐     ┌─────────────┐                                │
│  │  nats-node  │     │   pc-node   │                                │
│  │  (binary)   │     │  (binary)   │                                │
│  │             │     │             │                                │
│  │  NATS Hub   │     │  Runner +   │                                │
│  │  HA Cluster │     │  Platform   │                                │
│  │  Auth Mgmt  │     │  GUI        │                                │
│  └──────┬──────┘     └──────┬──────┘                                │
│         │                   │                                        │
│         └─────────┬─────────┘                                        │
│                   │                                                  │
│     Provided by: Self-hosted on laptop/server OR your cloud          │
└───────────────────┼──────────────────────────────────────────────────┘
                    │
                    │ leaf connections (auto-connect when hub available)
                    │
    ┌───────────────┼───────────────┬───────────────┐
    │               │               │               │
    ▼               ▼               ▼               ▼
┌────────┐     ┌────────┐     ┌────────┐     ┌────────┐
│ Svc A  │     │ Svc B  │     │ Svc C  │     │ Svc D  │
│        │     │        │     │        │     │        │
│ embed  │     │ embed  │     │ embed  │     │ embed  │
│ NATS   │     │ NATS   │     │ NATS   │     │ NATS   │
│ leaf   │     │ leaf   │     │ leaf   │     │ leaf   │
│        │     │        │     │        │     │        │
│ +GUI   │     │ +GUI   │     │ +GUI   │     │ +GUI   │
└────────┘     └────────┘     └────────┘     └────────┘

    All services import pkg/env
    All services get: embedded NATS + auth + config + secrets + GUI
    All services work offline, sync when hub available
```

---

## Components

### nats-node (Binary)

The NATS infrastructure. Runs as hub or forms HA cluster.

**What it provides:**
- NATS JetStream hub (central routing/persistence)
- Full auth lifecycle (none → token → nkey → jwt)
- KV buckets (services_registry, etc.)
- Leaf node listening for services to connect
- HA when running multiple instances

**How to run:**
```bash
# As hub (self-hosted)
NATS_PORT=4222 nats-node

# HA cluster (3 nodes)
NATS_PORT=4222 NATS_CLUSTER=nats://node2:6222,nats://node3:6222 nats-node
NATS_PORT=4223 NATS_CLUSTER=nats://node1:6222,nats://node3:6222 nats-node
NATS_PORT=4224 NATS_CLUSTER=nats://node1:6222,nats://node2:6222 nats-node
```

**Or use our cloud** - we run nats-node for you.

### pc-node (Binary)

The runner and platform GUI. Orchestrates everything via process-compose.

**What it provides:**
- Process-compose orchestration (run nats-node, services, etc.)
- Platform GUI (Via web UI) showing:
  - All registered services
  - Service dependency graph
  - Config/env requirements across fleet
  - Secret rotation status
  - Health/metrics
- Publishes process states to NATS

**How to run:**
```bash
# Run the platform locally
pc-node

# Opens GUI at http://localhost:3000
# Starts nats-node + your services via process-compose.yaml
```

### pkg/env (SDK)

The library developers import. **This is what makes it easy.**

**What developers get:**
```go
import "github.com/joeblew999/wellnown-env/pkg/env"

type Config struct {
    conf.Version
    Server struct {
        Host string `conf:"default:0.0.0.0:8080"`
        Port int    `conf:"default:8080"`
    }
    DB struct {
        Host     string `conf:"default:localhost"`
        Password string `conf:"mask,required"` // secret
    }
    Dependencies struct {
        AuthService string `conf:"service:joeblew999/auth-service"`
    }
}

func main() {
    mgr, _ := env.New("APP")
    defer mgr.Close()

    var cfg Config
    mgr.Parse(&cfg)

    // That's it. You now have:
    // 1. Config parsed from env vars
    // 2. Secrets resolved (ref+vault://, ref+file://, etc.)
    // 3. Embedded NATS leaf node (works offline)
    // 4. Registered to mesh (hub sees you)
    // 5. Ops GUI at http://localhost:3001 showing your config
    // 6. Can watch services you depend on
}
```

---

## What Developers Get For Free

### 1. Config From Struct

Define your config as a Go struct with tags:

```go
type Config struct {
    conf.Version
    Server struct {
        Host string `conf:"default:0.0.0.0:8080"`
        Port int    `conf:"default:8080,env:PORT"`
    }
    DB struct {
        Host     string `conf:"default:localhost"`
        Password string `conf:"mask,required"`
    }
    Cache struct {
        TTL time.Duration `conf:"default:5m"`
    }
}
```

**Tags:**
- `default:value` - default value
- `required` - must be set
- `mask` - it's a secret (masked in logs/GUI)
- `env:NAME` - custom env var name
- `service:org/repo` - dependency on another service

### 2. Secrets Resolved Automatically

Environment variables with `ref+` prefix are resolved via [helmfile/vals](https://github.com/helmfile/vals):

```bash
# Local development (file-based)
DB_PASSWORD=ref+file://./secrets/db_password.txt

# Production (Vault)
DB_PASSWORD=ref+vault://secret/prod/db#password

# 1Password
DB_PASSWORD=ref+op://Production/Database/password

# AWS Secrets Manager
DB_PASSWORD=ref+awssecrets://prod/db#password
```

**25+ backends supported.** Same code, different refs per environment.

### 3. Embedded NATS Leaf Node

Every service embeds a NATS node that:
- **Works offline** - full functionality without hub
- **Auto-syncs** - connects to hub when available
- **Persists locally** - data survives restarts (optional)
- **Auth included** - none/token/nkey/jwt lifecycle

```go
// Standalone (dev laptop, no hub)
mgr, _ := env.New("APP")

// Connects to hub (production)
// Just set NATS_HUB env var - code is identical
NATS_HUB=nats://hub.example.com:4222 ./myservice
```

### 4. Service Registration

Your config struct IS the registration schema. Zero duplication.

```go
// When you call mgr.Parse(&cfg), this happens:
// 1. Secrets resolved
// 2. Config parsed
// 3. Struct reflected to extract fields
// 4. Registration sent to NATS KV:

// NATS KV key: joeblew999.my-service.instance-abc123
{
    "github": {
        "org": "joeblew999",
        "repo": "my-service",
        "commit": "abc123",
        "tag": "v1.2.3",
        "branch": "main"
    },
    "instance": {
        "id": "abc123",
        "host": "10.0.0.5:8080",
        "started": "2024-01-15T10:30:00Z"
    },
    "fields": [
        {"path": "Server.Host", "type": "string", "default": "0.0.0.0:8080", "env_key": "APP_SERVER_HOST"},
        {"path": "Server.Port", "type": "int", "default": "8080", "env_key": "APP_SERVER_PORT"},
        {"path": "DB.Password", "type": "string", "required": true, "is_secret": true, "env_key": "APP_DB_PASSWORD"},
        {"path": "Dependencies.AuthService", "dependency": "joeblew999/auth-service"}
    ]
}
```

**The NATS registry becomes live documentation:**
- What env vars does any service need?
- What secrets are required?
- What version is running?
- Who depends on whom?

### 5. Service Discovery + Real-Time Updates

Watch services you depend on:

```go
// Get notified when auth-service changes
mgr.WatchService("joeblew999/auth-service", func(reg ServiceRegistration) {
    log.Printf("auth-service updated: %s at %s", reg.GitHub.Tag, reg.Instance.Host)

    // Update your internal client
    authClient.UpdateEndpoint(reg.Instance.Host)
})

// Get all instances of a service
instances, _ := mgr.GetService("joeblew999/auth-service")

// Get all registered services
services, _ := mgr.GetAllServices()
```

**Use cases:**
- Service A discovers Service B's endpoint changed → auto-reconnect
- Service B deployed new version → Service A sees new config requirements
- Service C scaled up → new instances immediately visible
- Service D config changed → dependent services react

No polling. Push-based via NATS KV watch.

### 6. Ops GUI For Free

Every service gets a Via web UI showing:
- Config values (secrets masked)
- Env var mappings
- Dependencies and their status
- NATS connection status
- Registration status

```go
mgr, _ := env.New("APP")
// GUI automatically available at http://localhost:3001
```

No Via code to write. The GUI is generated from your config struct.

### 7. Secret Rotation Notifications

Subscribe to secret rotation events:

```go
mgr.OnRotate(func(path string) {
    log.Printf("Secret rotated: %s", path)
    // Trigger graceful restart
    syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
})
```

The rotation service (or your cloud) publishes to `secrets.rotated.*` when secrets change.

---

## Design Principles

### Zero Duplication

The config struct IS:
1. Parsed by ardanlabs/conf for env vars + CLI flags
2. Sent to NATS KV as the registration schema
3. Used to generate the GUI
4. The single source of truth

No separate manifest. No Kubernetes ConfigMap schema. No documentation to maintain.

### Runner Independence

Same binary works everywhere:

| Runner | How it works |
|--------|--------------|
| **Docker/Compose** | Pass env vars via `environment:` |
| **Process Compose** | Pass env vars via `environment:` |
| **Kubernetes** | Pass env vars via ConfigMap/Secret |
| **systemd** | Pass env vars via `Environment=` |
| **Bare metal** | Export env vars or use `.env` file |

### GitHub Namespace = Service Namespace

GitHub `org/repo` IS the service identity:

```
services_registry/
├── joeblew999.api-gateway.instance-1
├── joeblew999.api-gateway.instance-2
├── joeblew999.auth-service.instance-1
└── somecorp.shared-lib.instance-1
```

Embedded via ldflags at build time.

### Offline-First

Services work completely offline:
- Embedded NATS stores data locally
- Syncs with hub when connectivity restored
- Perfect for edge, field devices, air-gapped environments

### Auth Lifecycle

Security grows with your needs:

| Phase | Auth Mode | Use Case |
|-------|-----------|----------|
| Dev | `none` | Fast iteration, no setup |
| Test/CI | `token` | Shared token via env var |
| Staging | `nkey` | NKey public/private keypairs |
| Production | `jwt` | Full NSC accounts with revocation |

All handled by nats-node. Services inherit auth automatically.

---

## Change Detection (Dev + CI/CD)

### Local Dev Check

```bash
wellknown-check --self

# Output:
# Your service (joeblew999/api-gateway) changes:
#   + APP_NEW_FEATURE (new required env)
#   - APP_OLD_FLAG (removed)
#   ~ APP_TIMEOUT (default: 5s → 10s)
#
# Services you depend on that changed:
#   ! joeblew999/auth-service added: AUTH_NEW_SCOPE (required)
```

### CI/CD Check

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
          wellknown-check \
            --repo ${{ github.repository }} \
            --pr-schema pr-schema.json \
            --check-deps \
            --check-consumers
```

### What Gets Checked

| Check | What it detects |
|-------|-----------------|
| `--self` | Changes in YOUR service's env/secret requirements |
| `--check-deps` | Changes in services YOU depend on |
| `--check-consumers` | Impact on services that depend on YOU |

Catch breaking changes BEFORE they hit production.

---

## Testing Strategy

**No mocks. No external services.** vals has built-in providers for testing:

```bash
# echo - returns the path as the value
APP_DB_PASSWORD=ref+echo://test-password

# file - reads from local file
APP_DB_PASSWORD=ref+file://./testdata/secrets/db_password.txt
```

```go
func TestConfig(t *testing.T) {
    t.Setenv("APP_DB_PASSWORD", "ref+echo://test-password")

    mgr, _ := env.New("APP")
    var cfg Config
    mgr.Parse(&cfg)

    assert.Equal(t, "test-password", cfg.DB.Password)
}
```

Same tests run locally and in CI with zero setup.

---

## Package Structure

```
wellknown-env/
├── pkg/
│   ├── env/                    # THE SDK
│   │   ├── env.go              # GetEnv, GetEnvInt, etc.
│   │   ├── vals.go             # ResolveEnvSecrets()
│   │   ├── manager.go          # Manager type, New(), Close()
│   │   ├── nats.go             # Embedded NATS leaf node
│   │   ├── auth.go             # Auth lifecycle
│   │   ├── parse.go            # Parse() - vals + conf + register
│   │   ├── fields.go           # Struct reflection for field extraction
│   │   ├── register.go         # NATS KV registration + heartbeat
│   │   ├── discovery.go        # WatchService, GetService
│   │   ├── rotation.go         # OnRotate subscription
│   │   ├── gui.go              # Via GUI auto-generation
│   │   ├── options.go          # Functional options
│   │   ├── errors.go           # Custom errors
│   │   └── registry/
│   │       ├── types.go        # ServiceRegistration, FieldInfo
│   │       └── github.go       # GitOrg, GitRepo ldflags vars
│   │
│   └── viatheme/               # Shared Via theme
│
├── examples/
│   ├── nats-node/              # NATS hub binary
│   │   ├── main.go
│   │   └── process-compose.yaml
│   │
│   ├── pc-node/                # Platform runner + GUI
│   │   ├── main.go
│   │   ├── pcview/             # PC viewer components
│   │   └── pc.yaml
│   │
│   ├── conf-only/              # ardanlabs/conf example
│   ├── vals-only/              # vals example
│   └── narun-hello/            # NATS microservice example
│
├── cmd/
│   └── wellknown-check/        # CI config validation tool
│       └── main.go
│
└── secrets/                    # Local secrets for dev
    ├── README.md
    └── *.example
```

---

## Implementation Phases

### Phase 1: SDK Foundation ✅ COMPLETE

**Files**: `pkg/env/`

1. ✅ Move auth.go from nats-node (full auth lifecycle)
2. ✅ Move NATS embedded setup from nats-node
3. ✅ Add ardanlabs/conf integration
4. ✅ vals integration already exists (vals.go)
5. ✅ Manager type wrapping it all

### Phase 2: Registration + Fields ✅ COMPLETE

1. ✅ ServiceRegistration type with GitHub identity
2. ✅ Field extraction via reflection (conf tags)
3. ✅ Support `service:org/repo` tag for dependencies
4. ✅ Heartbeat goroutine
5. ✅ ldflags for GitHub identity

### Phase 3: Discovery ✅ COMPLETE

1. ✅ WatchService() - NATS KV watcher
2. ✅ GetService() - list instances
3. ✅ GetAllServices() - list all
4. ✅ Push-based updates

### Phase 4: GUI ✅ COMPLETE

1. ✅ Composable Via pages (RegisterDashboardPage, RegisterConfigPage)
2. ✅ Show config values (masked secrets)
3. ✅ Show dependencies
4. ✅ Show NATS status
5. ✅ pcview package moved to pkg/env/pcview

### Phase 5: Secret Rotation ✅ COMPLETE

1. ✅ OnRotate() subscription (rotation.go)
2. ✅ PublishRotation() for rotation service
3. ✅ Manager.OnRotate() convenience method

### Phase 6: CI Tooling ✅ COMPLETE

1. ✅ wellknown-check CLI (cmd/wellknown-check/)
2. ✅ --schema-dump
3. ✅ --check-deps
4. ✅ --check-consumers
5. ✅ --self for local changes

### Phase 7: Refactor Examples ✅ COMPLETE

1. ✅ nats-node uses pkg/env
2. ✅ pc-node uses pkg/env
3. ✅ Both are thin wrappers around the SDK

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
func (m *Manager) GetAllServices() ([]ServiceRegistration, error)

// Secret rotation
func (m *Manager) OnRotate(fn func(path string)) error

// Access to internals
func (m *Manager) NC() *nats.Conn
func (m *Manager) KV() jetstream.KeyValue
func (m *Manager) GUIAddr() string

// Cleanup
func (m *Manager) Close() error
```

---

## Options

```go
// NATS topology
WithHub(url string)           // Connect to hub (default: standalone)
WithDataDir(path string)      // Persist data (default: in-memory)
WithPort(port int)            // NATS port (default: random)

// Registration
WithoutRegistration()         // Skip registration (for CLI tools)
WithoutHeartbeat()            // Skip heartbeat
WithHeartbeatInterval(sec)    // Custom interval (default: 10s)

// GUI
WithGUI(addr string)          // GUI address (default: :3001)
WithoutGUI()                  // Disable GUI
```

---

## Environment Variables

### Core

| Variable | Description |
|----------|-------------|
| `NATS_HUB` | Hub URL (empty = standalone) |
| `NATS_DATA` | Data directory (empty = in-memory) |
| `NATS_NAME` | Node name (default: random) |
| `NATS_PORT` | Client port (default: random) |
| `NATS_AUTH` | Auth mode: none, token, nkey, jwt |

### Build-time (ldflags)

| Variable | Description |
|----------|-------------|
| `GitOrg` | GitHub organization |
| `GitRepo` | GitHub repository name |
| `GitCommit` | Git commit hash |
| `GitTag` | Git tag/version |
| `GitBranch` | Git branch |

### vals Providers

| Provider | Required Variables |
|----------|-------------------|
| **1Password** | `OP_SERVICE_ACCOUNT_TOKEN` |
| **Vault** | `VAULT_ADDR`, `VAULT_TOKEN` |
| **AWS** | `AWS_PROFILE`, `AWS_DEFAULT_REGION` |
| **GCP** | `GOOGLE_APPLICATION_CREDENTIALS` |
| **Azure** | `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET` |

---

## Example Usage

### Basic Service

```go
package main

import (
    "log"
    "github.com/joeblew999/wellnown-env/pkg/env"
)

type Config struct {
    Server struct {
        Port int `conf:"default:8080"`
    }
    DB struct {
        URL      string `conf:"required"`
        Password string `conf:"mask,required"`
    }
}

func main() {
    mgr, err := env.New("APP")
    if err != nil {
        log.Fatal(err)
    }
    defer mgr.Close()

    var cfg Config
    if _, err := mgr.Parse(&cfg); err != nil {
        log.Fatal(err)
    }

    log.Printf("Server starting on :%d", cfg.Server.Port)
    log.Printf("GUI at %s", mgr.GUIAddr())

    // Your app logic here
}
```

### With Dependencies

```go
type Config struct {
    // ... your config ...

    Dependencies struct {
        AuthService    string `conf:"service:joeblew999/auth-service"`
        BillingService string `conf:"service:joeblew999/billing-api"`
    }
}

func main() {
    mgr, _ := env.New("APP")
    defer mgr.Close()

    var cfg Config
    mgr.Parse(&cfg)

    // Watch for auth-service updates
    mgr.WatchService("joeblew999/auth-service", func(reg ServiceRegistration) {
        log.Printf("auth-service now at: %s", reg.Instance.Host)
        // Update your auth client
    })

    // Your app logic
}
```

### Environment Files

```bash
# dev.env
APP_DB_URL=postgres://localhost/myapp
APP_DB_PASSWORD=ref+file://./secrets/db_password.txt

# prod.env
APP_DB_URL=postgres://prod-db.example.com/myapp
APP_DB_PASSWORD=ref+vault://secret/prod/db#password
NATS_HUB=nats://hub.example.com:4222
```

---

## The Platform Business Model

### Self-Hosted

Teams run nats-node + pc-node themselves:
- Full control
- Their infrastructure
- Free (open source)

### Cloud-Hosted

We run nats-node + pc-node for them:
- We manage the hub
- They just import pkg/env
- Their services connect to our cloud
- We provide: HA, monitoring, auth management, etc.

Same SDK, same code. Just different `NATS_HUB` value.

---

## Dependencies

```go
require (
    github.com/ardanlabs/conf/v3 v3.4.0
    github.com/helmfile/vals v0.37.8
    github.com/nats-io/nats-server/v2 v2.10.24
    github.com/nats-io/nats.go v1.38.0
    github.com/go-via/via v0.x.x
    github.com/google/uuid v1.6.0
)
```
