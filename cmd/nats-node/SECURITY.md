# NATS Security Lifecycle

| Phase | Mode | Usage |
|-------|------|-------|
| Dev | `none` | `task mesh` |
| Test | `token` | `task auth:token && task mesh` |
| Staging | `nkey` | `task auth:nkey && task mesh` |
| Prod | `jwt` | `task auth:jwt && task mesh` |

Reset: `task auth:clean`

Files: `.auth/` (gitignored)
