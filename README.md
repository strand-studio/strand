# Strand（牵发）

开源、自托管的 **AI 交易终端**：在你自己的机器上跑控制台与 API——密钥与资金不交给第三方托管。

**会交易的智能，但控制权在你手里。**

| 你想… | 去哪 |
|------|------|
| 看多所永续行情 / 标的详情 | **Markets**（无需登录） |
| 对比市价 / 限价 / Ladder / TWAP 再确认 | **Trade** |
| 跟随 Claw402 方向板自动交易 | **Autopilot** |
| 盯权益、持仓、硬闸与决策 | **Portfolio** |
| 调杠杆 / 日亏等硬风控 | **Strategy** |
| 公开对比 opt-in 实例 | **Leaderboard**（可匿名浏览） |

支持 **Hyperliquid**、**Binance**、**Bybit**、**OKX**、**Gate**、**Bitget** 等；AI 手续费钱包（Base USDC / Claw402）与交易保证金分离。

> 风险提示：加密与杠杆交易有本金损失风险。Strand 是软件工具，不是投资建议，不保证收益。

---

## 架构一览

```
┌─────────────────┐     /api/*      ┌──────────────────────┐
│  Web console    │ ───────────────▶│  Strand API (Go)     │
│  Vite / nginx   │                 │  cmd/strand          │
│  :3000          │                 │  :8080               │
└─────────────────┘                 └──────────┬───────────┘
                                               │
                    Markets public tickers / klines
                    Hyperliquid · CEX · Claw402 / Vergex
                    SQLite (default) or Postgres
```

本地开发时 Vite 把 `/api` 代理到 `localhost:8080`。Docker Compose 用 nginx 做同样的事。

---

## 快速开始（本地）

### 依赖

- **Go 1.26+**（SQLite 驱动需要 CGO / 系统 sqlite）
- **Node 20+** 与 **pnpm**（Web）
- 可选：能访问交易所公开行情与（若使用）OpenAI 兼容端点的出网；受限网络可设 `HTTPS_PROXY`

### 1. 初始化配置

```bash
go run ./cmd/strand config init
# 写入 ~/.strand/config.toml 与 ~/.strand/.env（含 JWT_SECRET / STRAND_API_TOKEN）
```

也可参考仓库内 [`config.example.toml`](./config.example.toml) 与 [`.env.example`](./.env.example)。

### 2. 启动 API

```bash
go run ./cmd/strand start
# 或: ./start.sh
# 或: make api
```

默认监听 `http://127.0.0.1:8080`。

### 3. 启动 Web

```bash
cd web
pnpm install
pnpm run dev
```

打开 `http://127.0.0.1:3000` → 注册 / 登录 → **Markets** 看行情，或按 [LAUNCH_CHECKLIST](./docs/LAUNCH_CHECKLIST.md) 走 Autopilot。

常用 Make 目标：`make help`、`make deps`、`make build`、`make web`、`make docker-up`。

---

## Docker

完整栈（API + Web）：

```bash
docker compose up --build
```

| 服务 | 地址 |
|------|------|
| Web 控制台 | http://127.0.0.1:3000 |
| API（也可直连） | http://127.0.0.1:8080 |

- 数据卷：`strand-data` → 容器内 `/data`（配置、SQLite、运行状态）
- 首次启动若无配置，入口脚本会执行 `strand config init --dir /data`
- 仅构建 API：`docker build -t strand-api -f docker/Dockerfile.backend .`
- 仅构建前端：`docker build -t strand-web -f docker/Dockerfile.frontend .`

出网需代理时，在 compose 或环境中设置 `HTTPS_PROXY` / `HTTP_PROXY`（见 [OPS_RUNBOOK](./docs/OPS_RUNBOOK.md)）。

---

## 产品能力（当前）

- **Markets**：多所 USDT 永续实时板；点击进入 `/markets/:exchange/:symbol` 详情（K 线、换算、表现、同所热门）
- **Trade**：人机共策下单规划
- **Autopilot**：Claw402 方向板 + Preflight + 硬风控闸门
- **Portfolio / Strategy / Leaderboard / Chat / Help / Settings**
- 公开接口示例：`GET /api/symbols`、`GET /api/klines`、`GET /api/competition`（无需登录）

顶栏与路由说明：[docs/WEB_TOP_NAV.md](./docs/WEB_TOP_NAV.md)  
产品介绍：[docs/PRODUCT_INTRO.md](./docs/PRODUCT_INTRO.md)  
控制台 API：[docs/WEB_API.md](./docs/WEB_API.md)  
运维：[docs/OPS_RUNBOOK.md](./docs/OPS_RUNBOOK.md)

---

## 仓库结构

```
cmd/strand/          CLI 入口（config / start / stop / status …）
internal/            API、交易、行情、策略、消息通道等
web/                 React 控制台（Vite + pnpm）
docker/              Dockerfile.backend / frontend、nginx、entrypoint
docs/                产品与运维文档
config.example.toml  配置示例
.env.example         密钥环境变量示例
```

---

## 安全与密钥

- **不要**把真实 `.env`、钱包私钥、交易所 API Key 提交进 Git（已在 `.gitignore`）
- Hyperliquid Agent 可交易、不可提现；CEX Key 建议关闭提现并做 IP 白名单
- 备份默认不含密钥；含密钥导出须离线保管（Settings / [OPS_RUNBOOK](./docs/OPS_RUNBOOK.md)）

---

## 许可证

[MIT](./LICENSE) © Strand
