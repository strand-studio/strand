# Strand（牵发）产品介绍

> 面向需要了解产品的人：价值、问题、收益、用法与后续规划。  
> 风险提示：加密与杠杆交易有本金损失风险；Strand 提供工具与风控闸门，不保证收益。

## 一句话

**Strand 是开源、自托管的 AI 交易终端：Autopilot 跟随 Claw402 方向板自动交易，Trade 让你先规划再下单；状态可跟进你日常用的社交工具——密钥与资金始终在你自己的机器上。**

品牌主张：**会交易的智能，但控制权在你手里。**

---

## 1. 产生了什么价值

Strand 把「看市场 → 做决策 → 下单执行 → 风控复盘 → 社交触达」收成一条可自托管的闭环，而不是又一个黑盒跟单机器人。

| 价值 | 说明 |
|------|------|
| **决策可执行** | AI 不只聊天看图，还能按策略周期开平仓，并留下完整决策与执行日志 |
| **资金可控** | 自托管、非托管；Hyperliquid Agent 可交易但不可提现；密钥留在本机 |
| **费用分账清晰** | AI 分析费（Claw402 / Base USDC 按次）与交易保证金分离，避免「AI 把保证金扣光」 |
| **风控在模型之外** | 日亏、回撤等硬闸独立于 AI，触发后暂停新开仓、保留持仓，并可推送到消息通道 |
| **两种交易节奏** | 全自动 Autopilot，或 Trade 对比市价/限价/Ladder/TWAP 后再确认 |
| **状态跟到 IM** | Telegram / Discord / Slack / 飞书 / 钉钉 / 微信 / 企微等：运维告警与可选 Agent 对话；Web Chat 只读问行情 |

---

## 2. 解决了什么问题

| 痛点 | Strand 怎么解 |
|------|----------------|
| 人工盯盘累、错过时机 | Autopilot 按周期读方向板 → Signal Lab / 清算结构校验 → K 线确认 → 自动执行 |
| SaaS 机器人要托管、不透明 | 开源自托管，账户与密钥在自己服务器 |
| 多模型 API Key 难管 | Claw402 按次付费；也支持 BYOK（自带模型） |
| AI「自信过头」乱开仓 | 置信度阈值、仓位/杠杆上限、日亏/回撤硬闸、Safe Mode |
| 半配置就上线亏钱 | 启动前 Preflight：AI 钱包、交易所余额、交易授权未就绪则拒绝启动 |
| 有时想自己拍板 | Trade：人机共策，审阅计划后再下单；Markets / Portfolio 可 Chart-to-order |
| 跑了不知道赚没赚净 | Portfolio（资产）权益曲线、费用、持仓与决策可追溯 |
| 人不在控制台就失联 | 消息通道推送硬闸 / Safe Mode / 崩溃告警；Telegram `/status` 可查运行中 Autopilot |
| 团队已在飞书/钉钉/企微协作 | Messaging 多平台绑定 + 对话 Agent（Claude Code / Codex 等），私有化触达 |

---

## 3. 为用户带来了什么

### 适合谁

- 想用 AI 辅助或全自动做合约/加密交易，但不愿把资金交给第三方托管的人
- 能自托管（本机 / Docker），会管钱包或 CEX API 的人
- 在「放手让 AI 跑」和「自己确认再下单」之间需要切换的人
- 希望告警与运维跟进 Telegram / 飞书 / 钉钉等日常 IM，而不是只盯网页的人

### 具体收益

1. **省盯盘时间**：候选标的与方向由方向板驱动，出场也跟板状态走（转向 / 中性 / 掉板则平）
2. **上手路径短**：充值 AI 费钱包（约 ≥ $1 Base USDC）→ 连接 Hyperliquid 或 CEX → 交易保证金就绪 → 一键启动
3. **多交易所**：Hyperliquid、Binance、Bybit、OKX、Gate、Bitget、KuCoin 等；另有 Lighter / Aster；HL 还可覆盖部分 HIP-3（`xyz:`）合成市场
4. **看得见的过程**：Portfolio（资产）看权益、持仓、硬闸与 AI 决策；Web Chat 用只读工具问行情与账户快照
5. **出事能感知**：Telegram / Discord / Slack 等可收启停、暂停、Safe Mode、崩溃告警；Messaging 可绑更多 IM
6. **可对比策略**：多 Autopilot 并行 + Leaderboard（排行榜），用同一规则比 PnL
7. **执行可规划**：Trade 对比市价 / 限价 / Ladder / TWAP；Chart-to-order 从行情进计划
8. **资源一眼清**：Settings → Resources 统一看模型、交易所、Agent、消息通道健康度

### 消息与社交：能做什么、不能做什么

对外请按下列边界表述（避免被理解成「聊天下单 SaaS」）：

| 通道 | 能做什么 | 不做 / 需限定 |
|------|----------|----------------|
| **Web Chat**（More → Chat） | 只读：行情、Vergex / 方向信号、账户快照、多场所覆盖 | **不下单**；下单请走 Trade |
| **控制台 Telegram Bot** | 交易运维告警；`/status` 查看运行中 Autopilot | 非命令闲聊不回复；不是行情 Copilot |
| **Notify（Discord / Slack）** | 硬闸暂停、Safe Mode、交易阻断、崩溃等告警推送 | 不是对话 Bot |
| **Messaging（多 IM）** | 绑 Telegram / 微信 / 企微 / 飞书 / Lark / 钉钉 / QQ / Discord / Slack / Matrix 等；可选对话 Agent（Claude Code、Codex、Copilot、Gemini、Kimi、OpenCode、Cursor）；可选告警 peer | **不是**产品级「一句话直接成交」；行情工具默认在 Web Chat，IM 侧靠 Agent 工作区扩展 |

一句话：**交易执行在你自己的机器上；状态与对话跟到你每天用的 IM。**

---

## 4. 产品怎么用（30 秒讲清）

```
注册登录 → Markets 看行情 → Autopilot 配 Claw402 / 交易所 → Preflight 绿灯启动
    ├─ Portfolio：盯权益、持仓、硬风控与决策日志
    └─ Trade：选市场 → 对比下单计划 → 确认执行
可选：Strategy 调硬闸 · Chat 问行情 · Leaderboard 对比
      · Settings → Messaging 绑 IM / Agent · Telegram 告警
```

更细的上线检查见 [LAUNCH_CHECKLIST.md](./LAUNCH_CHECKLIST.md)；顶栏各入口说明见 [WEB_TOP_NAV.md](./WEB_TOP_NAV.md)；小白边交易边学见 [CRYPTO_BEGINNER_LEARNING.md](./CRYPTO_BEGINNER_LEARNING.md)；控制台 API 见 [WEB_API.md](./WEB_API.md)。

---

## 5. 后续规划

当前仓库没有单独的正式 Roadmap 文档；M5 能力（会话、通知频道、简易回测、竞赛规则、Chart-to-order 等）已经上线。对外可按下列方向表述：

### 近期（巩固主路径）

- 把 Autopilot 启动与预检体验再压平，降低首单失败率
- 强化风控与告警（日亏/回撤、Safe Mode、多频道通知一致化）
- 策略工作室与简易回测向「更接近真实 Autopilot 回放」演进
- 通知侧更智能的状态回复与运维交互（持仓 / 权益快照扩到更多 IM）

### 中期（能力扩展）

- 更多场所与市场深度（已有 CEX / HL / 合成市场骨架）
- Trade 计划质量与执行类型（市价 / 限价 / Ladder / TWAP）持续打磨
- 竞赛与策略对比做成更清晰的「策略达尔文」机制
- Markets 页深化（更多场所实时价、迷你 K 线）；配置层已预留传统市场数据接口位（如 Alpaca / Twelve Data），按需求决定是否产品化
- IM 内「建议下单 → 二次确认 / 回 Trade 确认」若产品化，须带额度上限与审计日志；**不默认做成无确认市价成交**

### 原则不变

开源、自托管、密钥不出机、AI 费与交易金分离、硬风控在模型之外、社交触达不等于托管跟单。

---

## 6. 短版话术

### 30 字版

Strand：自托管 AI 交易终端。方向板自动交易，也能先规划再下单；告警跟到你的 IM，钱和钥匙都在你这边。

### 60 秒口播版

很多人想用 AI 交易，但怕两件事：钱被托管、机器人是黑盒。Strand（牵发）是你自己跑的开源终端：Autopilot 读 Claw402 方向板，校验后自动开平仓；想自己拍板就用 Trade 对比计划再下单。AI 分析费和交易保证金分开，日亏回撤有硬闸。Telegram、飞书、钉钉等可收告警，Web Chat 只读问行情——交易仍在你的机器上执行。适合愿意自托管、要可控自动交易的人——不是承诺暴利，是把决策、执行和风控放回你手里。

### 差异化三句

1. 不是托管跟单，是自托管执行；状态可跟进社交工具。
2. 不是纯 Prompt 瞎下单，是方向板 + 校验 + 硬风控。
3. 不是只能全自动：Autopilot 与 Trade 同一套账户体系；Chat 只读，下单走 Trade。

### 英文一句话（可选）

Open-source AI trading terminal you host yourself—Autopilot follows Claw402’s board; Trade lets you plan before you trade; alerts meet you in the apps you already use.

---

## 7. 按受众微调

| 对象 | 强调点 |
|------|--------|
| 交易员朋友 | 省盯盘、决策日志、硬风控、HL/CEX、Chart-to-order |
| 开发者 / 极客 | 开源、自托管、API、可审计、Messaging Agent 桥、可扩展策略 |
| 谨慎型用户 | 非托管、Agent 不可提现、Preflight、暂停/恢复、费用分账、Chat 不下单 |
| 进阶玩家 | 多策略竞赛、BYOK、Trade 执行计划、多场所、HIP-3 / Lighter / Aster |
| 团队 / 亚太私有化 | 飞书 / 钉钉 / 企微 / 微信触达、自托管部署、Resources 健康面板 |

---

## 8. 商业定位速记（对外可选）

| 项 | 建议表述 |
|----|----------|
| **品类** | Self-hosted AI Trading Terminal（非托管执行层，不是云端跟单 SaaS） |
| **主卖点** | 可控自动化 + 可审计闭环 + IM 运维触达 |
| **不主打** | 稳赚 / 暴利；「一句话聊天下单」 |
| **变现贴合点** | 开源获客；AI 用量（Claw402）；可选 Cloud 托管订阅；企业 IM 私有化实施 |
|