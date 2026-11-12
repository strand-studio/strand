# Security

- Report suspected vulnerabilities privately to the maintainers (GitHub Security Advisory or the contacts listed in the project).
- Never commit `.env`, private keys, or exchange credentials.
- Prefer CEX API keys **without withdrawal**, plus an IP allowlist for the Strand host.
- Hyperliquid Agent authorization is trading-only (cannot withdraw).
- Treat backups with `include_secrets=1` like a password vault dump.

Operational notes: [docs/OPS_RUNBOOK.md](./docs/OPS_RUNBOOK.md).
