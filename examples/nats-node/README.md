# nats-node

Embedded NATS JetStream server that runs as hub or leaf node.

Use Process Compose for each prcoess managment.

## Quick Start

```bash
# Run hub only
task hub

# Run full mesh (1 hub + 4 leaves)
task mesh

# Stop mesh
task mesh:down
```

## Testing

```bash
# List KV buckets
task test:kv

# Show registered services
task test:services

# Subscribe to all messages
task sub:all
```

## Cleanup

```bash
task clean       # Remove data directories
task clean:logs  # Remove log files
```

## Files

- `data/` - JetStream persistence (per node)
- `logs/` - Process logs (per node)
- `process-compose.log` - PC orchestrator log
