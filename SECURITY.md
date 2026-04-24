# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| main    | Yes       |

## Reporting a Vulnerability

This project handles authentication and OAuth2/OIDC flows, so security issues are taken seriously.

**Do not open a public GitHub issue for security vulnerabilities.**

Please report vulnerabilities privately via [GitHub's private vulnerability reporting](https://github.com/kuro48/idol-auth/security/advisories/new).

Include the following in your report:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (optional)

You will receive an acknowledgment within 72 hours and a status update within 7 days.

## Scope

In-scope:

- Authentication bypass or session fixation in `/v1/auth/*`
- Admin API authorization bypass in `/v1/admin/*`
- CSRF vulnerabilities in the consent flow
- Token leakage through logs or error responses
- SQL injection or other injection attacks

Out of scope:

- Vulnerabilities in Ory Hydra or Ory Kratos themselves (report to [ory/hydra](https://github.com/ory/hydra) or [ory/kratos](https://github.com/ory/kratos))
- Issues requiring physical access to the server
- Social engineering attacks
