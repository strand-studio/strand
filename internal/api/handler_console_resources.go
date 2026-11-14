package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"strand/internal/messaging"
	"strand/internal/store"

	"github.com/gin-gonic/gin"
)

func (s *Service) consoleResourcesOverview(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	rs := st.Resources()

	modelHealth, _ := rs.HealthMap(userID, store.ResourceKindModel)
	exchangeHealth, _ := rs.HealthMap(userID, store.ResourceKindExchange)
	platformHealth, _ := rs.HealthMap(userID, store.ResourceKindPlatform)
	agentHealth, _ := rs.HealthMap(userID, store.ResourceKindAgent)

	modelsOut := []gin.H{}
	var modelRows []*store.AIModel
	if list, err := st.AIModel().List(userID); err == nil {
		modelRows = list
		for _, m := range list {
			if m == nil {
				continue
			}
			row := sanitizeModel(m)
			for k, v := range store.AttachHealth(lookupModelHealth(modelHealth, m)) {
				row[k] = v
			}
			modelsOut = append(modelsOut, row)
		}
	}
	usage := buildResourcesUsage(st, modelRows)
	attachModelUsage(modelsOut, modelRows, usage)

	exchangesOut := []gin.H{}
	if list, err := st.Exchange().List(userID); err == nil {
		for _, ex := range list {
			if ex == nil {
				continue
			}
			row := sanitizeExchange(ex)
			for k, v := range store.AttachHealth(exchangeHealth[ex.ID]) {
				row[k] = v
			}
			exchangesOut = append(exchangesOut, row)
		}
	}

	platformsOut := []gin.H{}
	if list, err := st.Messaging().ListChannels(userID); err == nil {
		for _, ch := range list {
			if ch == nil {
				continue
			}
			row := channelPublic(ch)
			for k, v := range store.AttachHealth(platformHealth[ch.ID]) {
				row[k] = v
			}
			platformsOut = append(platformsOut, row)
		}
	}

	agentsOut := []gin.H{}
	ws, _ := st.Messaging().GetWorkspace(userID)
	configuredAgent := ""
	if ws != nil {
		configuredAgent = strings.TrimSpace(ws.AgentType)
	}
	for _, name := range messaging.Agents() {
		h := agentHealth[name]
		row := gin.H{
			"id":          name,
			"name":        name,
			"kind":        store.ResourceKindAgent,
			"configured":  configuredAgent != "" && strings.EqualFold(configuredAgent, name),
			"agent_type":  name,
		}
		for k, v := range store.AttachHealth(h) {
			row[k] = v
		}
		agentsOut = append(agentsOut, row)
	}

	defaults, err := rs.GetDefaults(userID)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}

	writeJSON(c, http.StatusOK, gin.H{
		"models":            modelsOut,
		"agents":            agentsOut,
		"platforms":         platformsOut,
		"exchanges":         exchangesOut,
		"defaults":          defaultsPublic(defaults),
		"usage":             usage,
		"healthy_only_hint": "Pickers should prefer enabled && last_test_ok (or untested for first-run).",
	})
}

// buildResourcesUsage summarizes AI fee wallet remaining (Claw402) and
// recorded spend (ai_charges). BYOK providers do not expose account quotas
// through Strand, so remaining tokens are not inventable there.
func buildResourcesUsage(st *store.Store, models []*store.AIModel) gin.H {
	out := gin.H{}
	if st == nil {
		return out
	}

	var walletAddr, modelHint string
	for _, m := range models {
		if m == nil || !strings.EqualFold(m.Provider, "claw402") {
			continue
		}
		if strings.TrimSpace(string(m.APIKey)) == "" {
			continue
		}
		addr, err := addressFromPrivateKey(string(m.APIKey))
		if err != nil || addr == "" {
			continue
		}
		walletAddr = addr
		modelHint = strings.TrimSpace(m.CustomModelName)
		if modelHint == "" {
			modelHint = "gpt-6-luna"
		}
		break
	}
	if walletAddr != "" {
		bal, status := cachedBaseUSDCBalance(walletAddr)
		balF, _ := strconv.ParseFloat(bal, 64)
		balanceStatus := status
		if status == "ok" {
			if balF > 0 {
				balanceStatus = "ready"
			} else {
				balanceStatus = "empty"
			}
		}
		daily, runway := store.EstimateRunway(balF, modelHint, 15)
		claw := gin.H{
			"wallet_address":  walletAddr,
			"balance_usdc":    bal,
			"balance_status":  balanceStatus,
			"est_daily_usd":   daily,
			"est_runway_days": runway,
			"model_hint":      modelHint,
		}
		out["claw402"] = claw
	}

	if charge := st.AICharge(); charge != nil {
		todayUSD, todayCalls, todayByModel, todayByProvider := charge.GetSummaryDetailed("today")
		weekUSD, weekCalls, weekByModel, weekByProvider := charge.GetSummaryDetailed("week")
		out["spend"] = gin.H{
			"today_usd":         todayUSD,
			"today_calls":       todayCalls,
			"week_usd":          weekUSD,
			"week_calls":        weekCalls,
			"by_model_today":    chargeBucketsPublic(todayByModel),
			"by_model_week":     chargeBucketsPublic(weekByModel),
			"by_provider_today": chargeBucketsPublic(todayByProvider),
			"by_provider_week":  chargeBucketsPublic(weekByProvider),
		}
	}
	return out
}

func chargeBucketsPublic(in map[string]store.ChargeBucket) gin.H {
	out := gin.H{}
	for k, v := range in {
		out[k] = gin.H{"usd": v.USD, "calls": v.Calls}
	}
	return out
}

func lookupChargeBucket(m gin.H, key string) (usd float64, calls int64, ok bool) {
	if m == nil || key == "" {
		return 0, 0, false
	}
	raw, exists := m[key]
	if !exists {
		return 0, 0, false
	}
	switch v := raw.(type) {
	case gin.H:
		usd, _ = toFloat64(v["usd"])
		calls, _ = toInt64(v["calls"])
		return usd, calls, true
	case map[string]any:
		usd, _ = toFloat64(v["usd"])
		calls, _ = toInt64(v["calls"])
		return usd, calls, true
	case float64:
		return v, 0, true
	default:
		return 0, 0, false
	}
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	case float32:
		return int64(n), true
	default:
		return 0, false
	}
}

// attachModelUsage writes remaining balance + per-model spend onto model rows.
func attachModelUsage(modelsOut []gin.H, modelRows []*store.AIModel, usage gin.H) {
	if len(modelsOut) == 0 {
		return
	}
	if claw, ok := usage["claw402"].(gin.H); ok {
		bal, _ := claw["balance_usdc"].(string)
		status, _ := claw["balance_status"].(string)
		addr, _ := claw["wallet_address"].(string)
		for _, row := range modelsOut {
			provider, _ := row["provider"].(string)
			if !strings.EqualFold(provider, "claw402") {
				continue
			}
			row["balance_usdc"] = bal
			row["balance_status"] = status
			if addr != "" {
				row["walletAddress"] = addr
			}
		}
	}

	spend, _ := usage["spend"].(gin.H)
	if spend == nil {
		return
	}
	byModelToday, _ := spend["by_model_today"].(gin.H)
	byModelWeek, _ := spend["by_model_week"].(gin.H)
	byProvToday, _ := spend["by_provider_today"].(gin.H)
	byProvWeek, _ := spend["by_provider_week"].(gin.H)

	for i, row := range modelsOut {
		var m *store.AIModel
		if i < len(modelRows) {
			m = modelRows[i]
		}
		id, _ := row["id"].(string)
		provider, _ := row["provider"].(string)
		customName, _ := row["customModelName"].(string)
		if m != nil {
			if customName == "" {
				customName = strings.TrimSpace(m.CustomModelName)
			}
			if provider == "" {
				provider = m.Provider
			}
			if id == "" {
				id = m.ID
			}
		}

		todayUSD, todayCalls, matched := resolveModelSpend(byModelToday, byProvToday, customName, provider, id)
		weekUSD, weekCalls, _ := resolveModelSpend(byModelWeek, byProvWeek, customName, provider, id)
		if !matched && todayUSD == 0 && weekUSD == 0 {
			continue
		}
		row["spend_today_usd"] = todayUSD
		row["spend_today_calls"] = todayCalls
		row["spend_week_usd"] = weekUSD
		row["spend_week_calls"] = weekCalls
	}
}

func resolveModelSpend(
	byModel, byProvider gin.H,
	customName, provider, id string,
) (usd float64, calls int64, ok bool) {
	for _, key := range []string{customName, id, provider} {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if u, c, found := lookupChargeBucket(byModel, key); found {
			return u, c, true
		}
		// Case-insensitive fallback for model name keys.
		for k := range byModel {
			if strings.EqualFold(k, key) {
				if u, c, found := lookupChargeBucket(byModel, k); found {
					return u, c, true
				}
			}
		}
	}
	for _, key := range []string{provider, id} {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if u, c, found := lookupChargeBucket(byProvider, key); found {
			return u, c, true
		}
		for k := range byProvider {
			if strings.EqualFold(k, key) {
				if u, c, found := lookupChargeBucket(byProvider, k); found {
					return u, c, true
				}
			}
		}
	}
	return 0, 0, false
}

func defaultsPublic(d *store.ResourceDefaults) gin.H {
	if d == nil {
		d = &store.ResourceDefaults{}
	}
	return gin.H{
		"default_model_id":    d.DefaultModelID,
		"default_agent_id":    d.DefaultAgentID,
		"default_exchange_id": d.DefaultExchangeID,
		"alert_platform_ids":  d.AlertIDs(),
		"updated_at":          d.UpdatedAt,
	}
}

func (s *Service) consoleResourcesDefaultsGet(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	d, err := st.Resources().GetDefaults(currentUserID(c))
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, defaultsPublic(d))
}

func (s *Service) consoleResourcesDefaultsPut(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	var body struct {
		DefaultModelID    string   `json:"default_model_id"`
		DefaultAgentID    string   `json:"default_agent_id"`
		DefaultExchangeID string   `json:"default_exchange_id"`
		AlertPlatformIDs  []string `json:"alert_platform_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	userID := currentUserID(c)
	d := &store.ResourceDefaults{
		UserID:            userID,
		DefaultModelID:    strings.TrimSpace(body.DefaultModelID),
		DefaultAgentID:    strings.TrimSpace(body.DefaultAgentID),
		DefaultExchangeID: strings.TrimSpace(body.DefaultExchangeID),
	}
	if body.AlertPlatformIDs == nil {
		existing, _ := st.Resources().GetDefaults(userID)
		if existing != nil {
			d.SetAlertIDs(existing.AlertIDs())
		} else {
			d.SetAlertIDs(nil)
		}
	} else {
		d.SetAlertIDs(body.AlertPlatformIDs)
	}
	if err := st.Resources().SaveDefaults(d); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	// Keep messaging workspace agent in sync when a scene default agent is chosen.
	if d.DefaultAgentID != "" {
		ws, err := st.Messaging().GetWorkspace(userID)
		if err == nil && ws != nil && !strings.EqualFold(ws.AgentType, d.DefaultAgentID) {
			ws.AgentType = d.DefaultAgentID
			_ = st.Messaging().SaveWorkspace(ws)
		}
	}
	writeJSON(c, http.StatusOK, defaultsPublic(d))
}

func (s *Service) consoleResourceTest(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	kind := strings.ToLower(strings.TrimSpace(c.Param("kind")))
	id := strings.TrimSpace(c.Param("id"))
	if kind == "" || id == "" {
		writeJSON(c, http.StatusBadRequest, gin.H{"ok": false, "error": "kind and id required"})
		return
	}
	userID := currentUserID(c)
	start := time.Now()
	var err error
	switch kind {
	case store.ResourceKindModel:
		err = s.testResourceModel(userID, id)
	case store.ResourceKindExchange:
		err = s.testResourceExchange(userID, id)
	case store.ResourceKindPlatform:
		err = s.testResourcePlatform(userID, id)
	case store.ResourceKindAgent:
		err = probeAgent(id)
	default:
		writeJSON(c, http.StatusBadRequest, gin.H{"ok": false, "error": "unknown kind (model|agent|platform|exchange)"})
		return
	}
	latency := time.Since(start).Milliseconds()
	ok := err == nil
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	_ = st.Resources().RecordTest(userID, kind, id, ok, latency, errMsg)
	if !ok {
		writeJSON(c, http.StatusBadRequest, gin.H{
			"ok":         false,
			"error":      errMsg,
			"latency_ms": latency,
			"tested_at":  time.Now().UTC().Format(time.RFC3339),
			"kind":       kind,
			"id":         id,
		})
		return
	}
	writeJSON(c, http.StatusOK, gin.H{
		"ok":         true,
		"latency_ms": latency,
		"tested_at":  time.Now().UTC().Format(time.RFC3339),
		"kind":       kind,
		"id":         id,
	})
}

func (s *Service) testResourceModel(userID, modelID string) error {
	st := s.eng.Store()
	m, err := st.AIModel().Get(userID, modelID)
	if err != nil || m == nil {
		return fmt.Errorf("model not found: %s", modelID)
	}
	return testModelCredentials(modelCredentialInput{
		Provider:        m.Provider,
		APIKey:          string(m.APIKey),
		CustomAPIURL:    m.CustomAPIURL,
		CustomModelName: m.CustomModelName,
	})
}

func (s *Service) testResourceExchange(userID, exchangeID string) error {
	st := s.eng.Store()
	ex, err := st.Exchange().GetByID(userID, exchangeID)
	if err != nil || ex == nil {
		return fmt.Errorf("exchange not found: %s", exchangeID)
	}
	body := exchangeWriteBody{
		ExchangeType:              ex.ExchangeType,
		APIKey:                    string(ex.APIKey),
		SecretKey:                 string(ex.SecretKey),
		Passphrase:                string(ex.Passphrase),
		Testnet:                   ex.Testnet,
		HyperliquidWalletAddr:     ex.HyperliquidWalletAddr,
		HyperliquidUnifiedAccount: ex.HyperliquidUnifiedAcct,
		AsterUser:                 ex.AsterUser,
		AsterSigner:               ex.AsterSigner,
		AsterPrivateKey:           string(ex.AsterPrivateKey),
		LighterWalletAddr:         ex.LighterWalletAddr,
		LighterPrivateKey:         string(ex.LighterPrivateKey),
		LighterAPIKeyPrivateKey:   string(ex.LighterAPIKeyPrivateKey),
		LighterAPIKeyIndex:        ex.LighterAPIKeyIndex,
	}
	return testExchangeCredentials(exchangeCredentialFromBody(userID, body))
}

func (s *Service) testResourcePlatform(userID, channelID string) error {
	st := s.eng.Store()
	ch, err := st.Messaging().GetChannel(userID, channelID)
	if err != nil || ch == nil {
		return fmt.Errorf("platform channel not found: %s", channelID)
	}
	return probePlatformChannel(ch)
}

// attachHealthFields merges health into an existing gin.H map.
func attachHealthFields(row gin.H, h *store.ResourceHealth) gin.H {
	for k, v := range store.AttachHealth(h) {
		row[k] = v
	}
	return row
}

// lookupModelHealth resolves health when tests were recorded under the
// provider / template id (e.g. "openai") before the row was created as
// "{userID}_openai".
func lookupModelHealth(health map[string]*store.ResourceHealth, m *store.AIModel) *store.ResourceHealth {
	if m == nil || health == nil {
		return nil
	}
	keys := []string{strings.TrimSpace(m.ID), strings.TrimSpace(m.Provider)}
	var fallback *store.ResourceHealth
	for _, key := range keys {
		if key == "" {
			continue
		}
		h := health[key]
		if h == nil {
			continue
		}
		if h.LastTestAt != nil {
			return h
		}
		if fallback == nil {
			fallback = h
		}
	}
	return fallback
}
