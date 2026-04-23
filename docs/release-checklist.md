# Release Checklist

## Config

- `APP_ENV=production`
- `APP_BASE_URL` is `https://...`
- `PORTAL_APP_URL` is `https://...`
- `KRATOS_BROWSER_URL` is `https://...`
- `HYDRA_BROWSER_URL` is `https://...`
- `SESSION_COOKIE_SECURE=true`
- `SESSION_COOKIE_DOMAIN` matches the real production cookie scope
- `TRUSTED_PROXIES` is set to the real ingress or load balancer CIDRs
- `LOG_LEVEL` is not `debug`
- `ADMIN_BOOTSTRAP_TOKEN` is rotated and stored in a secret manager (minimum 32 characters, not a known-weak value)
- Hydra `SECRETS_SYSTEM` (system secret) is at least 32 random characters and stored in a secret manager
- Kratos secret values (`SECRETS_DEFAULT`, `SECRETS_COOKIE`, `SECRETS_CIPHER`) are generated and stored in a secret manager
- `make render-production-config` has been run from the production environment file

## Infra

- TLS terminates only at approved ingress or load balancer layers
- `Caddy` or equivalent reverse proxy is the only public entrypoint
- `/v1/admin/*` is reachable only from operator networks or VPN
- `portal` is deployed and Kratos browser flows point to it
- Database backups are enabled and a restore drill has been run
- Redis is not exposed publicly
- Ory admin ports are not exposed publicly
- Runtime secrets come from a secret manager, not committed env files

## App Verification

- `go run ./cmd/configcheck` passes with production env injected
- `docker compose -f docker-compose.production.yml config` passes
- `go test ./...` passes on the release commit
- `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` reports no reachable vulnerabilities
- Local smoke or staging E2E passes for registration, MFA enrollment, OIDC, logout, identity re-enable, session revoke, and audit log read

## Operational Checks

- Alerting exists for app 5xx, auth failures, DB saturation, and container restarts
- Audit logs for admin actions are collected and retained
- Bootstrap token break-glass procedure is documented
- Rollback procedure is documented and tested
