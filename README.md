# wellnown-env

**env that's global, secure, and private** — because "it works on my machine" isn't a deployment strategy.

What if all your services that use services could be well-known and real-time?

## What Is This?

A Go library for config and secrets that doesn't make you cry:

- **Secrets**: Use `ref+file://`, `ref+vault://`, `ref+awssecrets://`, etc. — swap backends without changing code
- **Service Discovery**: Embedded NATS mesh so services find each other automatically
- **Local Dev**: File-based secrets. No Vault on your laptop. No mocking. Just files.

## Quick Start

```go
import "github.com/joeblew999/wellnown-env/pkg/env"

func main() {
    // Resolve any ref+ secrets (Vault, AWS, files, whatever)
    if err := env.ResolveEnvSecrets(); err != nil {
        log.Fatal(err)
    }

    // Now DB_PASSWORD contains the actual secret, not "ref+file://..."
    dbPass := os.Getenv("DB_PASSWORD")
}
```

## Local Development

Create secrets as files:

```bash
echo "mypassword" > secrets/db_password.txt
```

Reference them in your env:

```bash
DB_PASSWORD=ref+file://./secrets/db_password.txt
```

Done. Same code works with Vault in prod — just change the ref.

## Stack

- [helmfile/vals](https://github.com/helmfile/vals) — 25+ secret backends
- [nats-io/nats-server](https://github.com/nats-io/nats-server) — embedded mesh
- [process-compose](https://github.com/F1bonacc1/process-compose) — local orchestration

## Dog-Fooding

The infrastructure (nats-node, pc-node) uses this library. We eat our own cooking.
