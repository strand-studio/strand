package api

import (
	"net/http"

	"strand/internal/config"
	"strand/internal/settings"

	"github.com/gin-gonic/gin"
)

// Handler mounts the HTTP routes.
//
// - /api/v1/* : existing Strand CLI / operator API
// - /api/*    : web console API matching strand-web clients
func Handler(svc API, cfg settings.Settings) http.Handler {
	gin.SetMode(settings.NormalizeGinMode(cfg.API.Mode))
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(rateLimitMiddleware(newIPRateLimiter(20, 40)))

	concrete, _ := svc.(*Service)

	// --- v1 operator API (unchanged) ---
	base := config.ApiUri
	g := r.Group(base)
	g.GET("/health", healthHandler(svc))
	auth := g.Group("")
	auth.Use(requireAuth(cfg.API.Token))
	{
		auth.GET("/status", statusHandler(svc))
		auth.GET("/agents", agentsHandler(svc))
		auth.GET("/platforms", platformsHandler(svc))
		auth.GET("/traders", tradersListHandler(svc))
		auth.GET("/traders/:id", traderGetHandler(svc))
		auth.POST("/traders/:id/start", traderStartHandler(svc))
		auth.POST("/traders/:id/stop", traderStopHandler(svc))
	}
	r.GET("/api/health", healthHandler(svc))

	if concrete == nil {
		return r
	}

	// --- web console API under /api ---
	web := r.Group("/api")
	web.GET("/config", concrete.consoleConfig)
	web.GET("/crypto/config", concrete.consoleCryptoConfig)
	web.POST("/register", concrete.consoleRegister)
	web.POST("/login", concrete.consoleLogin)
	web.POST("/admin-login", concrete.consoleAdminLogin)
	web.POST("/logout", concrete.consoleLogout)

	// Public market catalog (no secrets) — powers Markets page without login.
	web.GET("/chains", concrete.consoleChains)
	web.GET("/symbols", concrete.consoleSymbols)
	web.GET("/klines", concrete.consoleKlines)

	// Public leaderboard (opt-in traders only; no keys / balances secrets).
	web.GET("/competition", concrete.consoleCompetition)
	web.GET("/top-traders", concrete.consoleCompetition)
	web.GET("/competition/rules", concrete.consoleCompetitionRulesGet)

	authed := web.Group("")
	authed.Use(requireUserAuth(cfg.API.Token))
	{
		authed.PUT("/user/password", concrete.consoleChangePassword)
		// traders
		authed.GET("/my-traders", concrete.consoleMyTraders)
		authed.GET("/traders", concrete.consolePublicTraders)
		authed.POST("/traders", concrete.consoleCreateTrader)
		authed.PUT("/traders/:id", concrete.consoleUpdateTrader)
		authed.PUT("/traders/:id/runtime", concrete.consoleTraderRuntime)
		authed.DELETE("/traders/:id", concrete.consoleDeleteTrader)
		authed.POST("/traders/:id/start", concrete.consoleStartTrader)
		authed.POST("/traders/:id/stop", concrete.consoleStopTrader)
		authed.POST("/traders/:id/pause", concrete.consolePauseTrader)
		authed.POST("/traders/:id/resume", concrete.consoleResumeTrader)
		authed.PUT("/traders/:id/competition", concrete.consoleTraderCompetition)
		authed.PUT("/traders/:id/prompt", concrete.consoleTraderPrompt)
		authed.GET("/traders/:id/config", concrete.consoleTraderConfig)
		authed.GET("/traders/:id/public-config", concrete.consoleTraderConfig)

		// runtime / dashboard
		authed.GET("/status", concrete.consoleTraderStatus)
		authed.GET("/account", concrete.consoleAccount)
		authed.GET("/positions", concrete.consolePositions)
		authed.GET("/decisions", concrete.consoleDecisions)
		authed.GET("/decisions/latest", concrete.consoleDecisionsLatest)
		authed.GET("/statistics", concrete.consoleStatistics)
		authed.GET("/statistics/full", concrete.consoleStatisticsFull)
		// models / exchanges
		authed.GET("/models", concrete.consoleModels)
		authed.PUT("/models", concrete.consoleUpdateModels)
		authed.POST("/models/test", concrete.consoleTestModel)
		authed.GET("/supported-models", concrete.consoleSupportedModels)
		authed.GET("/exchanges", concrete.consoleExchanges)
		authed.POST("/exchanges", concrete.consoleCreateExchange)
		authed.PUT("/exchanges", concrete.consoleUpdateExchanges)
		authed.POST("/exchanges/test", concrete.consoleTestExchange)
		authed.DELETE("/exchanges/:id", concrete.consoleDeleteExchange)
		authed.GET("/exchanges/account-state", concrete.consoleExchangeAccountState)
		authed.GET("/supported-exchanges", concrete.consoleSupportedExchanges)
		authed.GET("/prompt-templates", concrete.consolePromptTemplates)
		authed.GET("/server-ip", concrete.consoleServerIP)
		authed.GET("/backup/export", concrete.consoleBackupExport)
		authed.POST("/backup/import", concrete.consoleBackupImport)

		// strategies
		authed.GET("/strategies", concrete.consoleStrategies)
		authed.POST("/strategies", concrete.consoleStrategyCreate)
		authed.GET("/strategies/active", concrete.consoleStrategyActive)
		authed.GET("/strategies/default-config", concrete.consoleStrategyDefaultConfig)
		authed.GET("/strategies/:id", concrete.consoleStrategyGet)
		authed.PUT("/strategies/:id", concrete.consoleStrategyUpdate)
		authed.DELETE("/strategies/:id", concrete.consoleStrategyDelete)
		authed.POST("/strategies/:id/activate", concrete.consoleStrategyActivate)
		authed.POST("/strategies/:id/duplicate", concrete.consoleStrategyDuplicate)
		authed.POST("/strategies/:id/backtest", concrete.consoleStrategyBacktest)

		// Vergex / authenticated market extras
		authed.GET("/equity-history", concrete.consoleEquityHistory)
		authed.POST("/equity-history-batch", concrete.consoleEquityHistoryBatch)
		authed.GET("/positions/history", concrete.consolePositionsHistory)
		authed.GET("/vergex/direction-change/leaderboard", concrete.consoleVergexLeaderboard)
		authed.GET("/vergex/direction-change/current", concrete.consoleVergexCurrent)
		authed.GET("/vergex/direction-change/history", concrete.consoleVergexHistory)
		authed.GET("/vergex/cost-liquidation-heatmap", concrete.consoleVergexHeatmap)
		authed.GET("/vergex/holder-winrate-map", concrete.consoleVergexWinrate)
		authed.GET("/vergex/holder-winrate-map/holders", concrete.consoleVergexWinrateHolders)
		authed.GET("/vergex/flow-markets", concrete.consoleVergexFlowMarkets)

		// AI market chat
		authed.GET("/chat/tools", concrete.consoleChatTools)
		authed.POST("/chat/tools/test", concrete.consoleChatToolTest)
		authed.POST("/chat/completions", concrete.consoleChatCompletions)
		authed.POST("/chat/stream", concrete.consoleChatStream)
		authed.GET("/chat/sessions", concrete.consoleChatSessionsList)
		authed.POST("/chat/sessions", concrete.consoleChatSessionCreate)
		authed.GET("/chat/sessions/:id", concrete.consoleChatSessionGet)
		authed.PUT("/chat/sessions/:id", concrete.consoleChatSessionRename)
		authed.DELETE("/chat/sessions/:id", concrete.consoleChatSessionDelete)

		authed.POST("/launch/preflight", concrete.consoleLaunchPreflight)
		authed.GET("/traders/:id/preflight", concrete.consoleTraderPreflight)
		authed.POST("/onboarding/beginner", concrete.consoleBeginnerOnboarding)
		authed.GET("/onboarding/beginner/current", concrete.consoleBeginnerCurrent)
		authed.GET("/telegram", concrete.consoleTelegramGet)
		authed.POST("/telegram", concrete.consoleTelegramSave)
		authed.DELETE("/telegram/binding", concrete.consoleTelegramUnbind)
		authed.POST("/telegram/model", concrete.consoleTelegramSetModel)
		authed.GET("/notify/:type", concrete.consoleNotifyChannelGet)
		authed.POST("/notify/:type", concrete.consoleNotifyChannelSave)
		authed.DELETE("/notify/:type/binding", concrete.consoleNotifyChannelUnbind)
		authed.DELETE("/notify/:type", concrete.consoleNotifyChannelClear)

		authed.PUT("/competition/rules", concrete.consoleCompetitionRulesPut)

		// wallet / Hyperliquid connect
		authed.POST("/wallet/generate", concrete.consoleWalletGenerate)
		authed.POST("/wallet/validate", concrete.consoleWalletValidate)
		authed.GET("/wallet/balance", concrete.consoleWalletBalance)
		authed.POST("/wallet/export", concrete.consoleWalletExport)
		authed.GET("/hyperliquid/connect-config", concrete.consoleHyperliquidConnectConfig)
		authed.GET("/hyperliquid/account", concrete.consoleHyperliquidAccount)
		authed.GET("/hyperliquid/agent", concrete.consoleHyperliquidAgent)
		authed.POST("/hyperliquid/submit-exchange", concrete.consoleHyperliquidSubmitExchange)
	}

	return r
}
