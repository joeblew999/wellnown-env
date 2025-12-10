# pc-node

Reusable process-compose tasks + embedded Go example.

## Two Modes

**OS binary** (`task up`) - Control from outside via CLI. Full TUI, attach, logs.

**Embedded Go** (`task run`) - Process-compose runs *inside* your Go app.
Your code can programmatically start/stop/monitor processes. Enables Via/NATS
to control processes directly in Go - no shelling out.

## Run

```bash
# OS binary
task up           # Start detached
task attach       # Attach TUI
task down         # Stop

# Embedded Go (dog-foods itself)
task run          # Runs process-compose as library
```

## Reuse in other projects

```yaml
includes:
  pc:
    taskfile: path/to/Taskfile.pc.yml
```

Then: `task pc:up`, `task pc:list`, `task pc:down`, etc.

The Taskfile is binary-agnostic - works with OS binary or custom Go build.

## Environment Variables

Configuration follows a **single source of truth** pattern:

1. **Process-compose defines** environment variables in `pc.yaml`
2. **Go code reads** from environment using `pkg/env`

This eliminates hardcoded values and keeps configuration centralized.

### Available Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PC_ADDRESS` | `localhost` | Process-compose API host |
| `PC_PORT` | `8181` | Process-compose API port |
| `PC_URL` | - | Override full URL (takes precedence) |
| `VIA_ADDR` | `:3000` | Via web UI bind address |
| `VIA_PORT` | `3000` | Via web UI port |
| `VIA_HOST` | `localhost` | Via host for display URLs |
| `APP_NAME` | `wellnown-env` | Application name for dashboard |
| `LOG_LEVEL` | `info` | Logging level: debug, info, warn, error |
| `DEBUG` | `false` | Enable debug mode |

### Usage in Go

```go
import "github.com/joeblew999/wellnown-env/pkg/env"

// Load all config at once
cfg := env.LoadConfig()
fmt.Printf("PC API: %s\n", cfg.PCURL)
fmt.Printf("Via UI: %s\n", cfg.ViaURL)

// Or use individual helpers
pcURL := env.GetProcessComposeURL()    // http://localhost:8181
viaAddr := env.GetViaAddr()            // :3000
port := env.GetEnv("PC_PORT", "8181")  // with default
debug := env.GetEnvBool("DEBUG", false)
```

### Usage in pc.yaml

```yaml
# Global environment - inherited by all processes
environment:
  VIA_PORT: "3000"
  VIA_HOST: "localhost"
  PC_PORT: "8181"
  PC_ADDRESS: "localhost"
  APP_NAME: "pc-node"
  LOG_LEVEL: "info"
  DEBUG: "false"

processes:
  myapp:
    command: ./myapp
    # Process inherits all global env vars
```
