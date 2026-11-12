package api

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"strand/internal/store"

	"github.com/gin-gonic/gin"
)

func detectPublicIP() (ip, note string) {
	client := &http.Client{Timeout: 4 * time.Second}
	for _, url := range []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
	} {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
		resp.Body.Close()
		candidate := strings.TrimSpace(string(body))
		if net.ParseIP(candidate) != nil {
			return candidate, "detected"
		}
	}
	return "127.0.0.1", "could not detect public IP; using localhost"
}

func sanitizeModel(m *store.AIModel) gin.H {
	hasKey := strings.TrimSpace(string(m.APIKey)) != ""
	out := gin.H{
		"id":              m.ID,
		"name":            m.Name,
		"provider":        m.Provider,
		"enabled":         m.Enabled,
		"has_api_key":     hasKey,
		"customApiUrl":    m.CustomAPIURL,
		"customModelName": m.CustomModelName,
	}
	// Claw402 stores the EVM private key as api_key. Surface the derived
	// address so the UI can reopen an existing wallet instead of forcing
	// "create again" every time the modal opens.
	if strings.EqualFold(m.Provider, "claw402") && hasKey {
		if addr, err := addressFromPrivateKey(string(m.APIKey)); err == nil {
			out["walletAddress"] = addr
		}
	}
	return out
}

func sanitizeExchange(ex *store.Exchange) gin.H {
	return gin.H{
		"id":                            ex.ID,
		"exchange_type":                 ex.ExchangeType,
		"account_name":                  ex.AccountName,
		"name":                          ex.Name,
		"type":                          ex.Type,
		"enabled":                       ex.Enabled,
		"has_api_key":                   strings.TrimSpace(string(ex.APIKey)) != "",
		"has_secret_key":                strings.TrimSpace(string(ex.SecretKey)) != "",
		"has_passphrase":                strings.TrimSpace(string(ex.Passphrase)) != "",
		"testnet":                       ex.Testnet,
		"hyperliquidWalletAddr":         ex.HyperliquidWalletAddr,
		"hyperliquidUnifiedAccount":     ex.HyperliquidUnifiedAcct,
		"hyperliquidBuilderApproved":    ex.HyperliquidBuilderApproved,
		"has_hyperliquid_secret":        strings.TrimSpace(string(ex.SecretKey)) != "",
		"asterUser":                     ex.AsterUser,
		"asterSigner":                   ex.AsterSigner,
		"has_aster_private_key":         strings.TrimSpace(string(ex.AsterPrivateKey)) != "",
		"lighterWalletAddr":             ex.LighterWalletAddr,
		"has_lighter_private_key":       strings.TrimSpace(string(ex.LighterPrivateKey)) != "",
		"has_lighter_api_key_private_key": strings.TrimSpace(string(ex.LighterAPIKeyPrivateKey)) != "",
		"lighterApiKeyIndex":            ex.LighterAPIKeyIndex,
	}
}

func (s *Service) consoleModels(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, []any{})
		return
	}
	list, err := st.AIModel().List(currentUserID(c))
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, m := range list {
		out = append(out, sanitizeModel(m))
	}
	writeJSON(c, http.StatusOK, out)
}

func (s *Service) consoleSupportedModels(c *gin.Context) {
	// Template ids MUST equal provider names. The web modal selects by id and
	// matches configured rows by id/provider; empty ids made every open look
	// like a brand-new account.
	writeJSON(c, http.StatusOK, []gin.H{
		{"id": "claw402", "name": "Claw402", "provider": "claw402", "enabled": false, "type": "ai"},
		{"id": "deepseek", "name": "DeepSeek", "provider": "deepseek", "enabled": false, "type": "ai"},
		{"id": "openai", "name": "OpenAI", "provider": "openai", "enabled": false, "type": "ai"},
		{"id": "qwen", "name": "Qwen", "provider": "qwen", "enabled": false, "type": "ai"},
		{"id": "claude", "name": "Claude", "provider": "claude", "enabled": false, "type": "ai"},
		{"id": "gemini", "name": "Gemini", "provider": "gemini", "enabled": false, "type": "ai"},
		{"id": "grok", "name": "Grok", "provider": "grok", "enabled": false, "type": "ai"},
		{"id": "kimi", "name": "Kimi", "provider": "kimi", "enabled": false, "type": "ai"},
	})
}

type updateModelsBody struct {
	Models map[string]struct {
		Enabled         bool   `json:"enabled"`
		APIKey          string `json:"api_key"`
		CustomAPIURL    string `json:"custom_api_url"`
		CustomModelName string `json:"custom_model_name"`
	} `json:"models"`
}

func providerFromModelRef(idOrProvider string) string {
	ref := strings.TrimSpace(idOrProvider)
	if ref == "" {
		return ""
	}
	if i := strings.LastIndex(ref, "_"); i >= 0 && i+1 < len(ref) {
		return ref[i+1:]
	}
	return ref
}

func (s *Service) consoleTestModel(c *gin.Context) {
	st := s.eng.Store()
	userID := currentUserID(c)
	var body struct {
		Provider        string `json:"provider"`
		ModelID         string `json:"model_id"`
		APIKey          string `json:"api_key"`
		CustomAPIURL    string `json:"custom_api_url"`
		CustomModelName string `json:"custom_model_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	provider := strings.TrimSpace(body.Provider)
	if provider == "" {
		provider = providerFromModelRef(body.ModelID)
	}
	apiKey := strings.TrimSpace(body.APIKey)
	customURL := strings.TrimSpace(body.CustomAPIURL)
	customModel := strings.TrimSpace(body.CustomModelName)
	if apiKey == "" && st != nil && strings.TrimSpace(body.ModelID) != "" {
		if m, err := st.AIModel().Get(userID, body.ModelID); err == nil && m != nil {
			apiKey = string(m.APIKey)
			if customURL == "" {
				customURL = m.CustomAPIURL
			}
			if customModel == "" {
				customModel = m.CustomModelName
			}
			if provider == "" {
				provider = m.Provider
			}
		}
	}
	if err := testModelCredentials(modelCredentialInput{
		Provider:        provider,
		APIKey:          apiKey,
		CustomAPIURL:    customURL,
		CustomModelName: customModel,
	}); err != nil {
		writeJSON(c, http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

func (s *Service) consoleUpdateModels(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	var body updateModelsBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	// Live-test only rows that include a new/changed key so we never persist
	// invalid credentials. Unchanged keys (empty api_key) keep the stored secret.
	for idOrProvider, cfg := range body.Models {
		if strings.TrimSpace(cfg.APIKey) == "" {
			continue
		}
		provider := providerFromModelRef(idOrProvider)
		if existing, err := st.AIModel().Get(userID, idOrProvider); err == nil && existing != nil && existing.Provider != "" {
			provider = existing.Provider
		}
		if err := testModelCredentials(modelCredentialInput{
			Provider:        provider,
			APIKey:          cfg.APIKey,
			CustomAPIURL:    cfg.CustomAPIURL,
			CustomModelName: cfg.CustomModelName,
		}); err != nil {
			writeJSON(c, http.StatusBadRequest, gin.H{
				"ok":    false,
				"error": fmt.Sprintf("model %s: %v", idOrProvider, err),
			})
			return
		}
	}
	for idOrProvider, cfg := range body.Models {
		if err := st.AIModel().UpdateWithName(
			userID,
			idOrProvider,
			"",
			cfg.Enabled,
			cfg.APIKey,
			cfg.CustomAPIURL,
			cfg.CustomModelName,
		); err != nil {
			writeErr(c, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

func (s *Service) consoleExchanges(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, []any{})
		return
	}
	list, err := st.Exchange().List(currentUserID(c))
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, ex := range list {
		out = append(out, sanitizeExchange(ex))
	}
	writeJSON(c, http.StatusOK, out)
}

func (s *Service) consoleSupportedExchanges(c *gin.Context) {
	writeJSON(c, http.StatusOK, []gin.H{
		{"id": "", "exchange_type": "hyperliquid", "account_name": "", "name": "Hyperliquid", "type": "dex", "enabled": false, "chains": []string{"hyperliquid"}, "assets": []string{"USDC"}},
		{"id": "", "exchange_type": "binance", "account_name": "", "name": "Binance", "type": "cex", "enabled": false, "chains": []string{"binance"}, "assets": []string{"USDT", "USDC"}},
		{"id": "", "exchange_type": "bybit", "account_name": "", "name": "Bybit", "type": "cex", "enabled": false, "chains": []string{"bybit"}, "assets": []string{"USDT"}},
		{"id": "", "exchange_type": "okx", "account_name": "", "name": "OKX", "type": "cex", "enabled": false, "chains": []string{"okx"}, "assets": []string{"USDT"}},
		{"id": "", "exchange_type": "gate", "account_name": "", "name": "Gate", "type": "cex", "enabled": false, "chains": []string{"gate"}, "assets": []string{"USDT"}},
		{"id": "", "exchange_type": "bitget", "account_name": "", "name": "Bitget", "type": "cex", "enabled": false, "chains": []string{"bitget"}, "assets": []string{"USDT"}},
		{"id": "", "exchange_type": "kucoin", "account_name": "", "name": "KuCoin", "type": "cex", "enabled": false, "chains": []string{"kucoin"}, "assets": []string{"USDT"}},
		{"id": "", "exchange_type": "lighter", "account_name": "", "name": "Lighter", "type": "dex", "enabled": false, "chains": []string{"ethereum"}, "assets": []string{"USDC"}},
		{"id": "", "exchange_type": "aster", "account_name": "", "name": "Aster", "type": "dex", "enabled": false, "chains": []string{"ethereum"}, "assets": []string{"USDC"}},
	})
}

func (s *Service) consolePromptTemplates(c *gin.Context) {
	writeJSON(c, http.StatusOK, gin.H{
		"templates": []gin.H{
			{"name": "default"},
			{"name": "claw402_autopilot"},
		},
	})
}

func (s *Service) consoleServerIP(c *gin.Context) {
	ip, note := detectPublicIP()
	writeJSON(c, http.StatusOK, gin.H{
		"public_ip": ip,
		"message":   note,
	})
}

type exchangeWriteBody struct {
	ExchangeType               string `json:"exchange_type"`
	AccountName                string `json:"account_name"`
	Enabled                    bool   `json:"enabled"`
	APIKey                     string `json:"api_key"`
	SecretKey                  string `json:"secret_key"`
	Passphrase                 string `json:"passphrase"`
	Testnet                    bool   `json:"testnet"`
	HyperliquidWalletAddr      string `json:"hyperliquid_wallet_addr"`
	HyperliquidUnifiedAccount  bool   `json:"hyperliquid_unified_account"`
	HyperliquidBuilderApproved bool   `json:"hyperliquid_builder_approved"`
	AsterUser                  string `json:"aster_user"`
	AsterSigner                string `json:"aster_signer"`
	AsterPrivateKey            string `json:"aster_private_key"`
	LighterWalletAddr          string `json:"lighter_wallet_addr"`
	LighterPrivateKey          string `json:"lighter_private_key"`
	LighterAPIKeyPrivateKey    string `json:"lighter_api_key_private_key"`
	LighterAPIKeyIndex         int    `json:"lighter_api_key_index"`
}

func exchangeCredentialFromBody(userID string, body exchangeWriteBody) exchangeCredentialInput {
	return exchangeCredentialInput{
		ExchangeType:            body.ExchangeType,
		APIKey:                  body.APIKey,
		SecretKey:               body.SecretKey,
		Passphrase:              body.Passphrase,
		Testnet:                 body.Testnet,
		HyperliquidWalletAddr:   body.HyperliquidWalletAddr,
		HyperliquidUnifiedAcct:  body.HyperliquidUnifiedAccount,
		AsterUser:               body.AsterUser,
		AsterSigner:             body.AsterSigner,
		AsterPrivateKey:         body.AsterPrivateKey,
		LighterWalletAddr:       body.LighterWalletAddr,
		LighterPrivateKey:       body.LighterPrivateKey,
		LighterAPIKeyPrivateKey: body.LighterAPIKeyPrivateKey,
		LighterAPIKeyIndex:      body.LighterAPIKeyIndex,
		UserID:                  userID,
	}
}

func mergeExchangeCredentials(existing *store.Exchange, body *exchangeWriteBody) {
	if existing == nil || body == nil {
		return
	}
	if strings.TrimSpace(body.ExchangeType) == "" {
		body.ExchangeType = existing.ExchangeType
	}
	if strings.TrimSpace(body.APIKey) == "" {
		body.APIKey = string(existing.APIKey)
	}
	if strings.TrimSpace(body.SecretKey) == "" {
		body.SecretKey = string(existing.SecretKey)
	}
	if strings.TrimSpace(body.Passphrase) == "" {
		body.Passphrase = string(existing.Passphrase)
	}
	if strings.TrimSpace(body.HyperliquidWalletAddr) == "" {
		body.HyperliquidWalletAddr = existing.HyperliquidWalletAddr
	}
	if strings.TrimSpace(body.AsterUser) == "" {
		body.AsterUser = existing.AsterUser
	}
	if strings.TrimSpace(body.AsterSigner) == "" {
		body.AsterSigner = existing.AsterSigner
	}
	if strings.TrimSpace(body.AsterPrivateKey) == "" {
		body.AsterPrivateKey = string(existing.AsterPrivateKey)
	}
	if strings.TrimSpace(body.LighterWalletAddr) == "" {
		body.LighterWalletAddr = existing.LighterWalletAddr
	}
	if strings.TrimSpace(body.LighterPrivateKey) == "" {
		body.LighterPrivateKey = string(existing.LighterPrivateKey)
	}
	if strings.TrimSpace(body.LighterAPIKeyPrivateKey) == "" {
		body.LighterAPIKeyPrivateKey = string(existing.LighterAPIKeyPrivateKey)
	}
	if body.LighterAPIKeyIndex == 0 && existing.LighterAPIKeyIndex != 0 {
		body.LighterAPIKeyIndex = existing.LighterAPIKeyIndex
	}
}

func (s *Service) consoleTestExchange(c *gin.Context) {
	st := s.eng.Store()
	userID := currentUserID(c)
	var body struct {
		exchangeWriteBody
		ExchangeID string `json:"exchange_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if st != nil && strings.TrimSpace(body.ExchangeID) != "" {
		if existing, err := st.Exchange().GetByID(userID, body.ExchangeID); err == nil {
			mergeExchangeCredentials(existing, &body.exchangeWriteBody)
		}
	}
	if err := testExchangeCredentials(exchangeCredentialFromBody(userID, body.exchangeWriteBody)); err != nil {
		writeJSON(c, http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

func (s *Service) consoleCreateExchange(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	var body exchangeWriteBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(body.ExchangeType) == "" {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "exchange_type required"})
		return
	}
	if err := testExchangeCredentials(exchangeCredentialFromBody(userID, body)); err != nil {
		writeJSON(c, http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}
	name := strings.TrimSpace(body.AccountName)
	if name == "" {
		name = "Default"
	}
	id, err := st.Exchange().Create(
		userID,
		body.ExchangeType,
		name,
		body.Enabled,
		body.APIKey,
		body.SecretKey,
		body.Passphrase,
		body.Testnet,
		body.HyperliquidWalletAddr,
		body.HyperliquidUnifiedAccount,
		body.HyperliquidBuilderApproved,
		body.AsterUser,
		body.AsterSigner,
		body.AsterPrivateKey,
		body.LighterWalletAddr,
		body.LighterPrivateKey,
		body.LighterAPIKeyPrivateKey,
		body.LighterAPIKeyIndex,
	)
	if err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"id": id})
}

type updateExchangesBody struct {
	Exchanges map[string]exchangeWriteBody `json:"exchanges"`
}

func (s *Service) consoleUpdateExchanges(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	var body updateExchangesBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	for id, cfg := range body.Exchanges {
		existing, err := st.Exchange().GetByID(userID, id)
		if err != nil || existing == nil {
			writeJSON(c, http.StatusBadRequest, gin.H{"error": fmt.Sprintf("exchange not found: %s", id)})
			return
		}
		merged := cfg
		mergeExchangeCredentials(existing, &merged)
		if err := testExchangeCredentials(exchangeCredentialFromBody(userID, merged)); err != nil {
			writeJSON(c, http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
			return
		}
		if err := st.Exchange().Update(
			userID,
			id,
			true,
			cfg.APIKey,
			cfg.SecretKey,
			cfg.Passphrase,
			cfg.Testnet,
			merged.HyperliquidWalletAddr,
			merged.HyperliquidUnifiedAccount || cfg.HyperliquidUnifiedAccount,
			cfg.HyperliquidBuilderApproved || existing.HyperliquidBuilderApproved,
			merged.AsterUser,
			merged.AsterSigner,
			cfg.AsterPrivateKey,
			merged.LighterWalletAddr,
			cfg.LighterPrivateKey,
			cfg.LighterAPIKeyPrivateKey,
			merged.LighterAPIKeyIndex,
		); err != nil {
			writeErr(c, http.StatusBadRequest, err)
			return
		}
		if name := strings.TrimSpace(cfg.AccountName); name != "" {
			_ = st.Exchange().UpdateAccountName(userID, id, name)
		}
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

func (s *Service) consoleDeleteExchange(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	if err := st.Exchange().Delete(currentUserID(c), c.Param("id")); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}
