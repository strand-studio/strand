# Strand Web Console

React + TypeScript + Vite frontend for Strand.

For product overview, local/Docker setup, and CLI, see the root docs:

- [README.md](../README.md) (English, default)
- [README.zh-CN.md](../README.zh-CN.md) (中文)

```bash
pnpm install
pnpm run dev    # http://127.0.0.1:3000 — proxies /api → :8080
pnpm run build
pnpm test
```

Requires the Strand API running (`go run ./cmd/strand start` from the repo root).
