package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"strand/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const backupFormatVersion = 1

// consoleBackupExport exports user config (models/exchanges/strategies/traders).
// Secrets are omitted unless include_secrets=true (operator must confirm in UI).
func (s *Service) consoleBackupExport(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	includeSecrets := c.Query("include_secrets") == "1" || c.Query("include_secrets") == "true"

	models, err := st.AIModel().List(userID)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	exchanges, err := st.Exchange().List(userID)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	strategies, err := st.Strategy().List(userID)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	traders, err := st.Trader().List(userID)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}

	modelOut := make([]gin.H, 0, len(models))
	for _, m := range models {
		if m == nil {
			continue
		}
		row := sanitizeModel(m)
		if includeSecrets {
			row["apiKey"] = string(m.APIKey)
		}
		modelOut = append(modelOut, row)
	}

	exchangeOut := make([]gin.H, 0, len(exchanges))
	for _, ex := range exchanges {
		if ex == nil {
			continue
		}
		row := sanitizeExchange(ex)
		if includeSecrets {
			row["apiKey"] = string(ex.APIKey)
			row["secretKey"] = string(ex.SecretKey)
			row["passphrase"] = string(ex.Passphrase)
			row["asterPrivateKey"] = string(ex.AsterPrivateKey)
			row["lighterPrivateKey"] = string(ex.LighterPrivateKey)
			row["lighterApiKeyPrivateKey"] = string(ex.LighterAPIKeyPrivateKey)
		}
		exchangeOut = append(exchangeOut, row)
	}

	strategyOut := make([]gin.H, 0, len(strategies))
	for _, item := range strategies {
		if item == nil {
			continue
		}
		strategyOut = append(strategyOut, strategyPayload(item))
	}

	traderOut := make([]gin.H, 0, len(traders))
	for _, tr := range traders {
		if tr == nil {
			continue
		}
		traderOut = append(traderOut, gin.H{
			"id":                     tr.ID,
			"name":                   tr.Name,
			"ai_model_id":            tr.AIModelID,
			"exchange_id":            tr.ExchangeID,
			"strategy_id":            tr.StrategyID,
			"scan_interval_minutes":  tr.ScanIntervalMinutes,
			"is_cross_margin":        tr.IsCrossMargin,
			"show_in_competition":    tr.ShowInCompetition,
			"btc_eth_leverage":       tr.BTCETHLeverage,
			"altcoin_leverage":       tr.AltcoinLeverage,
			"trading_symbols":        tr.TradingSymbols,
			"custom_prompt":          tr.CustomPrompt,
			"override_base_prompt":   tr.OverrideBasePrompt,
			"system_prompt_template": tr.SystemPromptTemplate,
			"use_ai500":              tr.UseAI500,
			"use_oi_top":             tr.UseOITop,
		})
	}

	writeJSON(c, http.StatusOK, gin.H{
		"format":          "strand-backup",
		"version":         backupFormatVersion,
		"exported_at":     time.Now().UTC().Format(time.RFC3339),
		"include_secrets": includeSecrets,
		"user_id":         userID,
		"models":          modelOut,
		"exchanges":       exchangeOut,
		"strategies":      strategyOut,
		"traders":         traderOut,
	})
}

// consoleBackupImport restores strategies (always) and optionally model/exchange
// credentials when the backup was exported with include_secrets.
func (s *Service) consoleBackupImport(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	var body struct {
		Format         string          `json:"format"`
		Version        int             `json:"version"`
		IncludeSecrets bool            `json:"include_secrets"`
		Models         []gin.H         `json:"models"`
		Exchanges      []gin.H         `json:"exchanges"`
		Strategies     json.RawMessage `json:"strategies"`
		RestoreSecrets bool            `json:"restore_secrets"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if body.Format != "" && body.Format != "strand-backup" {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "unsupported backup format"})
		return
	}

	restoredStrategies := 0
	var strategies []struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Config      json.RawMessage `json:"config"`
	}
	if len(body.Strategies) > 0 {
		_ = json.Unmarshal(body.Strategies, &strategies)
	}
	for _, item := range strategies {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		cfg := "{}"
		if len(item.Config) > 0 {
			cfg = string(item.Config)
		}
		row := &store.Strategy{
			ID:          uuid.NewString(),
			UserID:      userID,
			Name:        name + " (restored)",
			Description: item.Description,
			Config:      cfg,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
		if err := st.Strategy().Create(row); err != nil {
			continue
		}
		restoredStrategies++
	}

	restoredModels := 0
	restoredExchanges := 0
	if body.RestoreSecrets && body.IncludeSecrets {
		for _, m := range body.Models {
			provider, _ := m["provider"].(string)
			apiKey, _ := m["apiKey"].(string)
			if strings.TrimSpace(provider) == "" || strings.TrimSpace(apiKey) == "" {
				continue
			}
			customURL, _ := m["customApiUrl"].(string)
			customModel, _ := m["customModelName"].(string)
			name, _ := m["name"].(string)
			id, _ := m["id"].(string)
			if id == "" {
				id = provider
			}
			if err := st.AIModel().UpdateWithName(userID, id, name, true, apiKey, customURL, customModel); err == nil {
				restoredModels++
			}
		}
		for _, ex := range body.Exchanges {
			exType, _ := ex["exchange_type"].(string)
			if strings.TrimSpace(exType) == "" {
				continue
			}
			accountName, _ := ex["account_name"].(string)
			apiKey, _ := ex["apiKey"].(string)
			secretKey, _ := ex["secretKey"].(string)
			passphrase, _ := ex["passphrase"].(string)
			testnet, _ := ex["testnet"].(bool)
			asterUser, _ := ex["asterUser"].(string)
			asterSigner, _ := ex["asterSigner"].(string)
			asterPK, _ := ex["asterPrivateKey"].(string)
			lighterAddr, _ := ex["lighterWalletAddr"].(string)
			lighterPK, _ := ex["lighterApiKeyPrivateKey"].(string)
			hlWallet, _ := ex["hyperliquidWalletAddr"].(string)
			lighterIdx := 0
			switch v := ex["lighterApiKeyIndex"].(type) {
			case float64:
				lighterIdx = int(v)
			case int:
				lighterIdx = v
			}
			if accountName == "" {
				accountName, _ = ex["name"].(string)
			}
			if accountName == "" {
				accountName = "Restored"
			}
			if _, err := st.Exchange().Create(
				userID, exType, accountName, true,
				apiKey, secretKey, passphrase, testnet,
				hlWallet, false, false,
				asterUser, asterSigner, asterPK,
				lighterAddr, "", lighterPK, lighterIdx,
			); err == nil {
				restoredExchanges++
			}
		}
	}

	writeJSON(c, http.StatusOK, gin.H{
		"ok":                  true,
		"restored_strategies": restoredStrategies,
		"restored_models":     restoredModels,
		"restored_exchanges":  restoredExchanges,
	})
}
