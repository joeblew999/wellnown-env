# wellknown-env

https://github.com/joeblew999/wellnown-env

You are an expert in golang, cloud secret stores, NATS, and Process Compose.

## Stack

- https://github.com/ardanlabs/conf/

- https://github.com/helmfile/vals

- https://github.com/nats-io/nats-server

- https://github.com/F1bonacc1/process-compose

- https://github.com/go-via/via
- https://github.com/go-via/via-plugin-picocss

- https://github.com/akhenakh/narun
  - Nats Micro Services


## Plan File

**Single Source of Truth**: `.claude/plans/plan.md`

Always keep the plan file in this repo so it's easy to pick up where we left off. Copy from `~/.claude/plans/` to `.claude/plans/plan.md` when planning is complete.

The TODO.md is the original discussion output from Claude chat - treat as dead code once the plan exists.

## Build Configuration

Use GOWORK off, so we dont need go.work files when we run the golang.

Use Nats Jetstream Embedded for the Server and Client base, so we have easy running and a locked down NATS.

## Reference Source Trick

When working with external libraries, clone them to `.src/` for easy reference:

```bash
mkdir -p .src && cd .src
git clone https://github.com/go-via/via-plugin-picocss.git
git clone https://github.com/go-via/via.git
# etc.
```

The `.src/` folder is gitignored. This lets you:
- Browse actual source code instead of guessing APIs
- Find working examples in `internal/example/`
- Understand how libraries work internally
- Copy patterns that work

Always check `.src/{lib}/internal/example/` for official usage patterns.



