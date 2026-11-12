# Strand Web 顶部导航说明

本文说明控制台顶栏（Header）各入口的作用，方便产品讲解、新手上手与协作对齐。  
顶栏固定在页面顶部；多数入口需登录，未登录点击会提示先登录。

布局参考 Hyperliquid App：左侧放交易主路径，次要入口收入 **More**；右侧只放偏好、钱包与账户。

**一致性约定：** 顶栏文案 = 页面 H1 = 路径英文 slug（见下表）。

相关文档：[PRODUCT_INTRO.md](./PRODUCT_INTRO.md) · [CRYPTO_BEGINNER_LEARNING.md](./CRYPTO_BEGINNER_LEARNING.md) · [LAUNCH_CHECKLIST.md](./LAUNCH_CHECKLIST.md) · [WEB_API.md](./WEB_API.md)

---

## 顶栏结构（从左到右）

```
[Strand Logo]
  Markets · Trade · Portfolio · Autopilot · Strategy · Leaderboard · More ▾
                                                          [语言] [主题] [钱包] [登录/账户]
```

| 区域 | 作用 |
|------|------|
| **Logo / 品牌** | 回到落地页 `/` |
| **主导航（左）** | 交易与运行主路径（见下表） |
| **More** | 次要入口：Chat、Help |
| **语言 / 主题** | 界面偏好 |
| **钱包连接** | Hyperliquid Agent 授权 |
| **登录 / 账户** | 未登录 Sign In；已登录设置 / 外链 / 退出 |

---

## 主导航各项（标题 · 路径一致）

| 显示名（en / zh） | 路径 | 旧路径（自动跳转） | 是否需登录 | 一句话作用 |
|-------------------|------|-------------------|------------|------------|
| **Markets** / 市场 | `/markets` | `/data` | 否 | 实时行情板；跳转 Trade 规划或 Chat 简报 |
| **Trade** / 交易 | `/trade` | `/frontier` · `/planning` | 是 | 人机共策：对比市价/限价/Ladder/TWAP 后再确认 |
| **Portfolio** / 资产 | `/portfolio` | `/dashboard` | 是 | 权益、持仓、硬闸与决策；可暂停 / Safe Mode |
| **Autopilot** / 自动驾驶 | `/autopilot` | `/traders` | 否（访客引导）/ 启动需登录 | 未登录看启动指南；登录后 Preflight → 启动 |
| **Strategy** / 策略 | `/strategy` | — | 是 | 硬风控、方向板规则、轻量回测后激活 |
| **Leaderboard** / 排行榜 | `/leaderboard` | `/competition` | 否 | 公开浏览 opt-in 排名；参赛需登录后启动 Autopilot |
| **More → Chat** / 对话 | `/chat` | — | 是 | 只读工具问行情；下单请走 Trade |
| **More → Help** / 帮助 | `/help` | `/faq` | 否 | 钱包 / 启动 / 控制台 / 告警 / 自托管指南 |
| **Settings** / 设置 | `/settings` | — | 是 | 账户菜单入口；AI / 交易所 / 备份 |

### 按使用意图怎么选

| 你想… | 去哪 |
|--------|------|
| 自己审计划再下单 | **Trade** |
| 看最新行情 | **Markets** |
| 看持仓 / 权益 / 决策 | **Portfolio** |
| 绑 AI、绑交易所、启动机器人 | **Autopilot** |
| 改杠杆 / 日亏硬闸 | **Strategy** |
| 横向对比多个 Trader | **Leaderboard** |
| 问一句「现在 BTC 怎样」 | **More → Chat** |
| 看帮助 | **More → Help** |

---

## 顶栏右侧控件

### 语言切换

- 选项：`EN` · `中文` · `ID`
- 持久化：浏览器 `localStorage`（键 `language`）

### 主题切换

- 顶栏：月亮 / 太阳一键切换浅色与深色
- 设置页可选：浅色 / 深色 / 跟随系统
- 持久化：`localStorage`（键 `strand.theme`）

### Hyperliquid 钱包

- 连接浏览器钱包并完成 Agent / 相关授权
- Agent 可交易、不可提现

### 账户菜单（已登录）

| 项 | 作用 |
|----|------|
| Settings | 打开 `/settings` |
| 外链 / 退出 | 官方链接与登出 |

---

## 推荐上手路径

```
落地页 → 注册 / 登录
  → Autopilot：绑 AI（Claw402）+ 交易所，Preflight 绿灯后启动
  → Portfolio：看权益、持仓、决策；必要时暂停
  → Trade：手动确认计划再下单
  → Strategy：调硬闸与方向板规则
```

---

## 实现备注

- 路由常量：`web/src/router/paths.ts`（`ROUTES.planning` 等仍作内部别名，但路径值为上表 canonical slug）。
- 文案：`tradeNav` / `pageTradeTitle` 等成对键保持同词；`leaderboardNav` 替代旧 `realtimeNav` 用于顶栏。
- 文案随语言变化：Markets / Trade / Portfolio / Autopilot 等在中文下为「市场 / 交易 / 资产 / 自动驾驶」等。
