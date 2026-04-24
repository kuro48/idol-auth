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

## Rollback

To roll back to a previous release:

1. Identify the target tag or commit:

```bash
git log --oneline
```

2. Check out the target version on the VPS:

```bash
cd /opt/idol-auth
git fetch
git checkout <tag-or-commit>
```

3. Re-deploy:

```bash
./scripts/run-nix-app.sh deploy-production .env.production
```

4. Verify:

```bash
curl -fsS https://<APP_HOSTNAME>/healthz
curl -fsS https://<APP_HOSTNAME>/readyz
```

If the migration introduced breaking schema changes, restore from the last backup before
re-deploying the older version (see Restore Drill above).

## Break-Glass: Bootstrap Token Rotation

Use this procedure when the `ADMIN_BOOTSTRAP_TOKEN` is suspected to be compromised or
needs emergency rotation.

1. Generate a new token:

```bash
openssl rand -hex 32
```

2. Update `.env.production` on the VPS:

```bash
vi /opt/idol-auth/.env.production
# Replace ADMIN_BOOTSTRAP_TOKEN=<old> with the new value
```

3. Restart only the app container (no downtime for Ory or Postgres):

```bash
cd /opt/idol-auth
docker compose -f docker-compose.production.yml --env-file .env.production \
  up -d --no-deps --build app
```

4. Confirm the old token is rejected and the new token works:

```bash
# Should return 401
curl -sf -H "Authorization: Bearer <old-token>" \
  https://<APP_HOSTNAME>/v1/admin/apps | jq .

# Should return 200
curl -sf -H "Authorization: Bearer <new-token>" \
  https://<APP_HOSTNAME>/v1/admin/apps | jq .
```

5. Store the new token in your secret manager and revoke the old one.

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

## Rate Limiter

The rate limiter uses an **in-memory sliding-window implementation** (no Redis).

- State resets on process restart
- State is not shared across replicas
- This is intentional for the current single-instance deployment on Sakura VPS

If you scale to multiple replicas in the future, replace `internal/http/ratelimit.go`
with a Redis-backed implementation and reconnect `REDIS_ADDR` / `REDIS_PASSWORD`
via the app environment.

## Monitoring Baseline

- container restarts
- host disk usage
- postgres volume growth
- `/readyz` failures
- repeated auth failures
- SMTP delivery failures

### Minimum Viable Alerting

Set up external uptime monitoring as a minimum before going to production:

1. Sign up for [UptimeRobot](https://uptimerobot.com) (free tier is sufficient).
2. Add an HTTP monitor for `https://<APP_HOSTNAME>/healthz` — interval 5 minutes.
3. Add an HTTP monitor for `https://<APP_HOSTNAME>/readyz` — interval 5 minutes.
4. Configure email or Slack alerts on status change.

To verify health from the VPS itself after a deploy:

```bash
make check-health AUTH_URL=http://localhost:8080
```

Docker-level health status:

```bash
docker compose -f docker-compose.production.yml ps
```

The `app` container reports `healthy` / `unhealthy` based on the `/readyz` probe
(30s interval, 5s timeout, 3 retries, 10s start period).
