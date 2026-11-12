package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"strand/internal/store"

	"github.com/gin-gonic/gin"
)

const (
	minAIFeeUSDC    = 1.0
	minTradingUSDC  = 12.0
	accountProbeTO  = 18 * time.Second
)

type launchCheck struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"` // ok | failed | warning | skipped
	Code         string  `json:"code,omitempty"`
	Message      string  `json:"message,omitempty"`
	Required     float64 `json:"required,omitempty"`
	Actual       float64 `json:"actual,omitempty"`
	Asset        string  `json:"asset,omitempty"`
	Address      string  `json:"address,omitempty"`
	ExchangeID   string  `json:"exchange_id,omitempty"`
	ExchangeType string  `json:"exchange_type,omitempty"`
}

func annotateExchange(check launchCheck, ex *store.Exchange) launchCheck {
	if ex != nil {
		check.ExchangeID = ex.ID
		check.ExchangeType = ex.ExchangeType
	}
	return check
}

type launchPreflightResult struct {
	Ready         bool          `json:"ready"`
	Checks        []launchCheck `json:"checks"`
	MinAIFeeUSDC  float64       `json:"min_ai_fee_usdc"`
	MinTradingUSDC float64      `json:"min_trading_usdc"`
	CheckedAt     string        `json:"checked_at"`
}

type exchangeAccountProbe struct {
	ExchangeID       string
	Status           string
	DisplayBalance   string
	Asset            string
	TotalEquity      float64
	AvailableBalance float64
	ErrorCode        string
	ErrorMessage     string
}

func (s *Service) consoleExchangeAccountState(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, gin.H{"states": gin.H{}})
		return
	}
	userID := currentUserID(c)
	list, err := st.Exchange().List(userID)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}

	states := make(map[string]gin.H, len(list))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, ex := range list {
		if ex == nil {
			continue
		}
		wg.Add(1)
		go func(ex *store.Exchange) {
			defer wg.Done()
			probe := s.probeExchangeAccount(userID, ex)
			mu.Lock()
			states[ex.ID] = gin.H{
				"exchange_id":       probe.ExchangeID,
				"status":            probe.Status,
				"display_balance":   probe.DisplayBalance,
				"asset":             probe.Asset,
				"total_equity":      probe.TotalEquity,
				"available_balance": probe.AvailableBalance,
				"checked_at":        time.Now().UTC().Format(time.RFC3339),
				"error_code":        probe.ErrorCode,
				"error_message":     probe.ErrorMessage,
			}
			mu.Unlock()
		}(ex)
	}
	wg.Wait()
	writeJSON(c, http.StatusOK, gin.H{"states": states})
}

func (s *Service) probeExchangeAccount(userID string, ex *store.Exchange) exchangeAccountProbe {
	out := exchangeAccountProbe{
		ExchangeID: ex.ID,
		Asset:      exchangeAsset(ex.ExchangeType),
	}
	if !ex.Enabled {
		out.Status = "disabled"
		out.ErrorCode = "disabled"
		out.ErrorMessage = "Exchange is disabled"
		return out
	}
	if !exchangeHasCredentials(ex) {
		out.Status = "missing_credentials"
		out.ErrorCode = "missing_credentials"
		out.ErrorMessage = "Finish exchange credentials before verifying"
		return out
	}

	tr, err := buildCredentialExchangeTrader(exchangeCredentialFromStore(userID, ex))
	if err != nil {
		out.Status = "missing_credentials"
		out.ErrorCode = "missing_credentials"
		out.ErrorMessage = err.Error()
		return out
	}

	done := make(chan struct{})
	var bal map[string]any
	var balErr error
	go func() {
		bal, balErr = tr.GetBalance()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(accountProbeTO):
		out.Status = "unavailable"
		out.ErrorCode = "timeout"
		out.ErrorMessage = "Exchange balance probe timed out"
		return out
	}
	if balErr != nil {
		msg := balErr.Error()
		out.ErrorMessage = msg
		low := strings.ToLower(msg)
		switch {
		case strings.Contains(low, "permission") || strings.Contains(low, "forbidden") || strings.Contains(low, "not authorized"):
			out.Status = "permission_denied"
			out.ErrorCode = "permission_denied"
		case strings.Contains(low, "invalid") || strings.Contains(low, "api key") || strings.Contains(low, "signature") || strings.Contains(low, "auth"):
			out.Status = "invalid_credentials"
			out.ErrorCode = "invalid_credentials"
		default:
			out.Status = "unavailable"
			out.ErrorCode = "unavailable"
		}
		return out
	}

	total, _ := floatFromBalanceMap(bal, "totalEquity", "total_equity", "totalWalletBalance")
	avail, _ := floatFromBalanceMap(bal, "availableBalance", "available_balance")
	out.Status = "ok"
	out.TotalEquity = total
	out.AvailableBalance = avail
	out.DisplayBalance = fmt.Sprintf("%.2f %s", total, out.Asset)
	return out
}

func (s *Service) consoleLaunchPreflight(c *gin.Context) {
	var body struct {
		AIModelID  string `json:"ai_model_id"`
		ExchangeID string `json:"exchange_id"`
		StrategyID string `json:"strategy_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	result := s.runLaunchPreflight(currentUserID(c), body.AIModelID, body.ExchangeID, body.StrategyID)
	writeJSON(c, http.StatusOK, result)
}

func (s *Service) consoleTraderPreflight(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := c.Param("id")
	cfg, err := st.Trader().GetFullConfig(userID, id)
	if err != nil || cfg == nil || cfg.Trader == nil {
		writeJSON(c, http.StatusBadRequest, gin.H{
			"error":     "trader not found",
			"error_key": "trader.not_found",
		})
		return
	}
	result := s.runLaunchPreflight(
		userID,
		cfg.Trader.AIModelID,
		cfg.Trader.ExchangeID,
		cfg.Trader.StrategyID,
	)
	writeJSON(c, http.StatusOK, result)
}

func (s *Service) runLaunchPreflight(userID, aiModelID, exchangeID, strategyID string) launchPreflightResult {
	result := launchPreflightResult{
		Checks:         make([]launchCheck, 0, 7),
		MinAIFeeUSDC:   minAIFeeUSDC,
		MinTradingUSDC: minTradingUSDC,
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	st := s.eng.Store()
	if st == nil {
		result.Checks = append(result.Checks, launchCheck{
			ID:      "ai_model",
			Status:  "failed",
			Code:    "store_unavailable",
			Message: "Database is not ready",
		})
		return result
	}

	// --- AI model ---
	var model *store.AIModel
	aiModelID = strings.TrimSpace(aiModelID)
	if aiModelID == "" {
		result.Checks = append(result.Checks, launchCheck{
			ID: "ai_model", Status: "failed", Code: "missing_ai_model",
			Message: "Select an AI model before launching",
		})
	} else if m, err := st.AIModel().Get(userID, aiModelID); err != nil || m == nil {
		// provider fallback
		if list, lerr := st.AIModel().List(userID); lerr == nil {
			for _, item := range list {
				if item == nil {
					continue
				}
				if item.Provider == aiModelID || strings.HasSuffix(item.ID, "_"+aiModelID) {
					m = item
					err = nil
					break
				}
			}
		}
		if err != nil || m == nil {
			result.Checks = append(result.Checks, launchCheck{
				ID: "ai_model", Status: "failed", Code: "ai_model_not_found",
				Message: "AI model not found — open Config → AI connections",
			})
		} else {
			model = m
		}
	} else {
		model = m
	}

	if model != nil {
		if !model.Enabled {
			result.Checks = append(result.Checks, launchCheck{
				ID: "ai_model", Status: "failed", Code: "ai_model_disabled",
				Message: "AI model is disabled",
			})
		} else if strings.TrimSpace(string(model.APIKey)) == "" {
			result.Checks = append(result.Checks, launchCheck{
				ID: "ai_model", Status: "failed", Code: "ai_model_no_key",
				Message: "AI model has no API key / wallet key configured",
			})
		} else {
			result.Checks = append(result.Checks, launchCheck{
				ID: "ai_model", Status: "ok", Message: "AI model ready",
			})
		}
	}

	// --- AI wallet / funds (Claw402 only) ---
	if model != nil && strings.EqualFold(model.Provider, "claw402") {
		addr, err := addressFromPrivateKey(string(model.APIKey))
		if err != nil {
			result.Checks = append(result.Checks,
				launchCheck{ID: "ai_wallet", Status: "failed", Code: "invalid_wallet_key", Message: "Claw402 wallet key is invalid"},
				launchCheck{ID: "ai_wallet_funds", Status: "failed", Code: "invalid_wallet_key", Message: "Cannot check AI fee balance without a valid wallet"},
			)
		} else {
			result.Checks = append(result.Checks, launchCheck{
				ID: "ai_wallet", Status: "ok", Address: addr, Message: "Claw402 wallet derived",
			})
			balStr, balErr := fetchBaseUSDCBalance(addr)
			actual := 0.0
			if balErr == nil {
				actual, _ = strconv.ParseFloat(strings.TrimSpace(balStr), 64)
			}
			if balErr != nil {
				result.Checks = append(result.Checks, launchCheck{
					ID: "ai_wallet_funds", Status: "warning", Code: "balance_unavailable",
					Message:  "Could not read Base USDC balance; deposits may still be pending",
					Required: minAIFeeUSDC, Actual: actual, Asset: "USDC", Address: addr,
				})
			} else if actual < minAIFeeUSDC {
				result.Checks = append(result.Checks, launchCheck{
					ID: "ai_wallet_funds", Status: "failed", Code: "insufficient_ai_fee",
					Message:  fmt.Sprintf("AI fee wallet needs at least %.0f USDC on Base (have %.2f)", minAIFeeUSDC, actual),
					Required: minAIFeeUSDC, Actual: actual, Asset: "USDC", Address: addr,
				})
			} else {
				result.Checks = append(result.Checks, launchCheck{
					ID: "ai_wallet_funds", Status: "ok",
					Message:  "AI fee wallet funded",
					Required: minAIFeeUSDC, Actual: actual, Asset: "USDC", Address: addr,
				})
			}
		}
	} else {
		result.Checks = append(result.Checks,
			launchCheck{ID: "ai_wallet", Status: "skipped", Message: "Not a Claw402 wallet model"},
			launchCheck{ID: "ai_wallet_funds", Status: "skipped", Message: "AI fee wallet check skipped for API-key models"},
		)
	}

	// --- Strategy (optional) ---
	strategyID = strings.TrimSpace(strategyID)
	if strategyID == "" {
		result.Checks = append(result.Checks, launchCheck{
			ID: "strategy", Status: "skipped", Message: "Using default / active strategy",
		})
	} else if _, err := st.Strategy().Get(userID, strategyID); err != nil {
		result.Checks = append(result.Checks, launchCheck{
			ID: "strategy", Status: "failed", Code: "strategy_not_found",
			Message: "Selected strategy was not found",
		})
	} else {
		result.Checks = append(result.Checks, launchCheck{
			ID: "strategy", Status: "ok", Message: "Strategy found",
		})
	}

	// --- Exchange ---
	exchangeID = strings.TrimSpace(exchangeID)
	var ex *store.Exchange
	if exchangeID == "" {
		result.Checks = append(result.Checks,
			launchCheck{ID: "exchange_config", Status: "failed", Code: "missing_exchange", Message: "Select an exchange account before launching"},
			launchCheck{ID: "exchange_account", Status: "failed", Code: "missing_exchange", Message: "Exchange account check skipped"},
			launchCheck{ID: "exchange_funds", Status: "failed", Code: "missing_exchange", Message: "Trading funds check skipped"},
			launchCheck{ID: "exchange_trade_auth", Status: "failed", Code: "missing_exchange", Message: "Trading auth check skipped"},
		)
	} else {
		var err error
		ex, err = st.Exchange().GetByID(userID, exchangeID)
		if err != nil || ex == nil {
			result.Checks = append(result.Checks,
				launchCheck{ID: "exchange_config", Status: "failed", Code: "exchange_not_found", Message: "Exchange account not found"},
				launchCheck{ID: "exchange_account", Status: "failed", Code: "exchange_not_found", Message: "Cannot verify a missing exchange"},
				launchCheck{ID: "exchange_funds", Status: "failed", Code: "exchange_not_found", Message: "Cannot verify funds for a missing exchange"},
				launchCheck{ID: "exchange_trade_auth", Status: "failed", Code: "exchange_not_found", Message: "Trading auth check skipped"},
			)
		} else if !ex.Enabled {
			result.Checks = append(result.Checks,
				annotateExchange(launchCheck{ID: "exchange_config", Status: "failed", Code: "exchange_disabled", Message: "Exchange account is disabled"}, ex),
				annotateExchange(launchCheck{ID: "exchange_account", Status: "failed", Code: "exchange_disabled", Message: "Enable the exchange account first"}, ex),
				annotateExchange(launchCheck{ID: "exchange_funds", Status: "failed", Code: "exchange_disabled", Message: "Trading funds check skipped"}, ex),
				annotateExchange(launchCheck{ID: "exchange_trade_auth", Status: "failed", Code: "exchange_disabled", Message: "Trading auth check skipped"}, ex),
			)
		} else if !exchangeHasCredentials(ex) {
			result.Checks = append(result.Checks,
				annotateExchange(launchCheck{ID: "exchange_config", Status: "failed", Code: "missing_credentials", Message: "Finish exchange credentials / wallet authorization"}, ex),
				annotateExchange(launchCheck{ID: "exchange_account", Status: "failed", Code: "missing_credentials", Message: "Exchange credentials incomplete"}, ex),
				annotateExchange(launchCheck{ID: "exchange_funds", Status: "failed", Code: "missing_credentials", Message: "Trading funds check skipped"}, ex),
				annotateExchange(launchCheck{ID: "exchange_trade_auth", Status: "failed", Code: "missing_credentials", Message: "Trading auth check skipped"}, ex),
			)
		} else {
			result.Checks = append(result.Checks, annotateExchange(launchCheck{
				ID: "exchange_config", Status: "ok", Message: "Exchange credentials present",
			}, ex))
			probe := s.probeExchangeAccount(userID, ex)
			asset := probe.Asset
			if probe.Status != "ok" {
				result.Checks = append(result.Checks,
					annotateExchange(launchCheck{
						ID: "exchange_account", Status: "failed", Code: probe.ErrorCode,
						Message: firstNonEmpty(probe.ErrorMessage, "Exchange account verification failed"),
						Asset:   asset,
					}, ex),
					annotateExchange(launchCheck{
						ID: "exchange_funds", Status: "failed", Code: probe.ErrorCode,
						Message: "Cannot verify trading funds until the exchange account connects",
						Asset:   asset, Required: minTradingUSDC,
					}, ex),
					annotateExchange(launchCheck{
						ID: "exchange_trade_auth", Status: "failed", Code: probe.ErrorCode,
						Message: "Trading auth check skipped until the exchange account connects",
					}, ex),
				)
			} else {
				result.Checks = append(result.Checks, annotateExchange(launchCheck{
					ID: "exchange_account", Status: "ok",
					Message: "Exchange account reachable",
					Actual:  probe.TotalEquity, Asset: asset,
				}, ex))
				avail := probe.AvailableBalance
				if avail <= 0 {
					avail = probe.TotalEquity
				}
				if avail < minTradingUSDC {
					result.Checks = append(result.Checks, annotateExchange(launchCheck{
						ID: "exchange_funds", Status: "failed", Code: "insufficient_trading_funds",
						Message:  fmt.Sprintf("Trading account needs at least %.0f %s available (have %.2f)", minTradingUSDC, asset, avail),
						Required: minTradingUSDC, Actual: avail, Asset: asset,
					}, ex))
				} else {
					result.Checks = append(result.Checks, annotateExchange(launchCheck{
						ID: "exchange_funds", Status: "ok",
						Message:  "Trading funds sufficient",
						Required: minTradingUSDC, Actual: avail, Asset: asset,
					}, ex))
				}
				result.Checks = append(result.Checks, annotateExchange(s.probeExchangeTradeAuth(userID, ex), ex))
			}
		}
	}

	result.Ready = true
	for _, check := range result.Checks {
		if check.Status == "failed" {
			result.Ready = false
			break
		}
	}
	return result
}

func (s *Service) probeExchangeTradeAuth(userID string, ex *store.Exchange) launchCheck {
	check := launchCheck{ID: "exchange_trade_auth"}
	if ex == nil {
		check.Status = "failed"
		check.Code = "missing_exchange"
		check.Message = "Exchange account missing"
		return check
	}
	tr, err := buildCredentialExchangeTrader(exchangeCredentialFromStore(userID, ex))
	if err != nil {
		check.Status = "failed"
		check.Code = "missing_credentials"
		check.Message = err.Error()
		return check
	}
	checker, ok := tr.(interface{ CheckTradingAuthorization() error })
	if !ok {
		// CEX venues without an explicit auth probe: credentials + balance already checked.
		check.Status = "ok"
		check.Message = "Trade auth probe not required for this venue (credentials verified)"
		return check
	}
	done := make(chan error, 1)
	go func() { done <- checker.CheckTradingAuthorization() }()
	select {
	case err := <-done:
		if err != nil {
			check.Status = "failed"
			check.Code = "permission_denied"
			check.Message = "Trading authorization failed: " + err.Error()
			return check
		}
		check.Status = "ok"
		check.Message = "Trading authorization confirmed"
		return check
	case <-time.After(accountProbeTO):
		check.Status = "warning"
		check.Code = "timeout"
		check.Message = "Trading authorization probe timed out; start may still fail if agent is unauthorized"
		return check
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func describeLaunchPreflightFailures(result launchPreflightResult) string {
	parts := make([]string, 0, len(result.Checks))
	for _, check := range result.Checks {
		if check.Status == "failed" && strings.TrimSpace(check.Message) != "" {
			parts = append(parts, check.Message)
		}
	}
	if len(parts) == 0 {
		return "Launch preflight failed"
	}
	return strings.Join(parts, " ")
}
