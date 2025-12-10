# Local Secrets

This directory stores secrets for local development using `ref+file://` syntax.

**IMPORTANT**: This directory is gitignored. Never commit real secrets.

## Usage

1. Create secret files in this directory:
   ```
   echo "my-db-password" > secrets/db_password.txt
   echo '{"token": "abc123"}' > secrets/config.json
   ```

2. Reference them in your environment:
   ```bash
   # .env or process-compose.yaml
   DB_PASSWORD=ref+file://./secrets/db_password.txt
   CONFIG_TOKEN=ref+file://./secrets/config.json#/token
   ```

3. The `env.ResolveEnvSecrets()` call resolves them at startup.

## Example Files

Create these for local development:

| File | Content | Env Var |
|------|---------|---------|
| `db_password.txt` | Database password | `DB_PASSWORD=ref+file://./secrets/db_password.txt` |
| `api_key.txt` | API key | `API_KEY=ref+file://./secrets/api_key.txt` |
| `config.json` | JSON config | `TOKEN=ref+file://./secrets/config.json#/token` |

## Production

In production, switch to your secret store:

```bash
# Vault
DB_PASSWORD=ref+vault://secret/db#password

# AWS Secrets Manager
DB_PASSWORD=ref+awssecrets://prod/db#password

# 1Password
DB_PASSWORD=ref+op://Production/Database/password
```

Same code, different refs. The `vals` library handles resolution transparently.
