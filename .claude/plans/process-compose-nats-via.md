# Task: Root-Level Process-Compose for NATS + Via

**Goal**: Create `process-compose.yaml` at repo root to orchestrate NATS + Via together using the OS-level `process-compose` CLI.

**File to create**: `process-compose.yaml`

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    process-compose up                        │
│                                                              │
│  ┌──────────────────┐        ┌──────────────────┐           │
│  │   nats-embedded  │        │    via-embed     │           │
│  │   (hub, port     │        │   (port 3000)    │           │
│  │    4222)         │        │                  │           │
│  └──────────────────┘        └──────────────────┘           │
│         ▲                           │                        │
│         │      depends_on           │                        │
│         └───────────────────────────┘                        │
└─────────────────────────────────────────────────────────────┘
```

---

## The YAML

```yaml
version: "0.5"

processes:
  nats:
    command: go run main.go
    working_dir: examples/nats-embedded
    environment:
      - NATS_NAME=hub
      - NATS_PORT=4222
      - GOWORK=off
    readiness_probe:
      exec:
        command: "nc -z localhost 4222"
      initial_delay_seconds: 2

  via:
    command: go run main.go
    working_dir: examples/via-embed
    environment:
      - VIA_THEME=purple
      - GOWORK=off
    depends_on:
      nats:
        condition: process_healthy
```

---

## Usage

```bash
process-compose up        # Start both
process-compose up --tui  # With TUI
process-compose down      # Stop
```

---

## What This Demonstrates

1. NATS starts first as the hub (config store)
2. Via waits for NATS to be healthy before starting
3. Both run together, orchestrated by process-compose

Future: Via will connect to NATS and read/write `VIA_THEME` from NATS KV for real-time config propagation.
