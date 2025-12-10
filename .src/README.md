# Reference Source Code

Clone external libraries here for easy reference during development.

These are **NOT dependencies** - just for reading source code and finding examples.

## Quick Start

```bash
cd .src
task clone:all    # Clone all reference repos
task              # List available repos
```

## Why?

When working with external libraries, having the source locally lets you:
- Browse actual source code instead of guessing APIs
- Find working examples in `internal/example/`
- Understand how libraries work internally
- Copy patterns that work

## Available Repos

| Repo | Description |
|------|-------------|
| `via` | Via web framework |
| `via-plugin-picocss` | Via PicoCSS plugin |
| `browser` | gost-dom headless browser for testing |
| `process-compose` | Process orchestrator |
| `nats-server` | NATS messaging server |
| `narun` | NATS micro services |
| `conf` | Ardanlabs config library |
| `vals` | Secret store values (helmfile) |

## Commands

```bash
task clone:all        # Clone all repos
task clone:<name>     # Clone specific repo (e.g., task clone:via)
task update:all       # Pull latest for all repos
task clean            # Remove all cloned repos
```

## Tips

Always check `{repo}/internal/example/` or `{repo}/examples/` for official usage patterns.
