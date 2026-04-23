# Operations

## Target Shape

- 1 Linux VM
- Docker Engine
- `docker-compose.production.yml`
- `Caddy` as the only public ingress
- `app`, `portal`, `hydra public` reachable through `443`
- `hydra admin`, `kratos admin`, `postgres`, `redis` remain internal-only

## Initial Provisioning

1. Run `scripts/provision-sakura-vps.sh` once as `root`, or reproduce the same steps manually.
2. Create `.env.production` from `.env.production.example`.
3. Fill real hostnames, secret values, and SMTP settings.
4. Run:

```bash
cd /opt/idol-auth
./scripts/run-nix-app.sh deploy-production .env.production
```

## Production Deploy

```bash
cd /opt/idol-auth
git pull
./scripts/run-nix-app.sh deploy-production .env.production
```

The deploy script performs:

- production config render
- config validation
- compose validation
- image build or refresh
- stack rollout
- readiness wait on `app`

Without `Nix`, `scripts/deploy-production.sh` still works directly.

## Health Checks

- Internal app liveness: `http://127.0.0.1:8080/healthz`
- Internal app readiness: `http://127.0.0.1:8080/readyz`
- Public API: `https://auth.example.com/healthz`
- Public portal: `https://accounts.example.com/`
- Hydra public: `https://login.example.com/.well-known/openid-configuration`

## Backup

Manual backup:

```bash
cd /opt/idol-auth
./scripts/run-nix-app.sh backup-postgres .env.production
```

Files are written to `backups/` by default.

Systemd units are provided:

- `deploy/systemd/idol-auth.service`
- `deploy/systemd/idol-auth-backup.service`
- `deploy/systemd/idol-auth-backup.timer`

Install them to `/etc/systemd/system/`, then run:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now idol-auth.service
sudo systemctl enable --now idol-auth-backup.timer
```

## Restore Drill

1. Copy a backup archive to the host.
2. Stop the stack.
3. Recreate or empty the target database.
4. Restore with:

```bash
gunzip -c backups/idol_auth_<timestamp>.sql.gz | docker compose --env-file .env.production -f docker-compose.production.yml exec -T postgres psql -U idol -d idol_auth
```

5. Start the stack.
6. Verify login, MFA, and OIDC.

## Secrets Handling

- Do not commit `.env.production`.
- Prefer a secret manager over plain files.
- Rotate:
  - `ADMIN_BOOTSTRAP_TOKEN`
  - `HYDRA_SYSTEM_SECRET`
  - `KRATOS_SECRETS_*`
  - database and redis credentials
  - SMTP credentials

## Admin Access

- Keep `/v1/admin/*` behind VPN, bastion, or IP allowlist.
- Do not expose Ory admin ports publicly.
- Use session auth for read-only admin access.
- Keep bootstrap token for break-glass writes only.

## Monitoring Baseline

- container restarts
- host disk usage
- postgres volume growth
- `/readyz` failures
- repeated auth failures
- SMTP delivery failures
