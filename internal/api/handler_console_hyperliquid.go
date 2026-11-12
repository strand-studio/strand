package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const hyperliquidAPIURL = "https://api.hyperliquid.xyz"

func hlInfo(payload any) (json.RawMessage, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Post(hyperliquidAPIURL+"/info", "application/json", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hyperliquid info HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (s *Service) consoleHyperliquidAccount(c *gin.Context) {
	address := strings.TrimSpace(c.Query("address"))
	if address == "" || !strings.HasPrefix(strings.ToLower(address), "0x") {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "address required"})
		return
	}

	raw, err := hlInfo(map[string]string{"type": "clearinghouseState", "user": address})
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	var state struct {
		MarginSummary struct {
			AccountValue    string `json:"accountValue"`
			TotalMarginUsed string `json:"totalMarginUsed"`
			TotalNtlPos     string `json:"totalNtlPos"`
		} `json:"marginSummary"`
		Withdrawable string `json:"withdrawable"`
		AssetPositions []struct {
			Position struct {
				UnrealizedPnl string `json:"unrealizedPnl"`
			} `json:"position"`
		} `json:"assetPositions"`
	}
	_ = json.Unmarshal(raw, &state)

	spotUSDC := 0.0
	spotAvail := 0.0
	if spotRaw, spotErr := hlInfo(map[string]string{"type": "spotClearinghouseState", "user": address}); spotErr == nil {
		var spot struct {
			Balances []struct {
				Coin     string `json:"coin"`
				Total    string `json:"total"`
				Hold     string `json:"hold"`
				Available string `json:"available"`
			} `json:"balances"`
		}
		if json.Unmarshal(spotRaw, &spot) == nil {
			for _, b := range spot.Balances {
				if strings.EqualFold(b.Coin, "USDC") {
					spotUSDC, _ = strconv.ParseFloat(b.Total, 64)
					if b.Available != "" {
						spotAvail, _ = strconv.ParseFloat(b.Available, 64)
					} else {
						hold, _ := strconv.ParseFloat(b.Hold, 64)
						spotAvail = spotUSDC - hold
					}
				}
			}
		}
	}

	accountValue, _ := strconv.ParseFloat(state.MarginSummary.AccountValue, 64)
	withdrawable, _ := strconv.ParseFloat(state.Withdrawable, 64)
	marginUsed, _ := strconv.ParseFloat(state.MarginSummary.TotalMarginUsed, 64)
	unrealized := 0.0
	for _, p := range state.AssetPositions {
		u, _ := strconv.ParseFloat(p.Position.UnrealizedPnl, 64)
		unrealized += u
	}

	writeJSON(c, http.StatusOK, gin.H{
		"address":           address,
		"accountValue":      accountValue,
		"withdrawable":      withdrawable,
		"totalMarginUsed":   marginUsed,
		"unrealizedPnl":     unrealized,
		"openPositions":     len(state.AssetPositions),
		"spotUsdc":          spotUSDC,
		"spotUsdcAvailable": spotAvail,
		"updatedAt":         time.Now().UnixMilli(),
	})
}

func (s *Service) consoleHyperliquidAgent(c *gin.Context) {
	address := strings.TrimSpace(c.Query("address"))
	if address == "" {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "address required"})
		return
	}

	raw, err := hlInfo(map[string]string{"type": "extraAgents", "user": address})
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	var agents []struct {
		Name       string `json:"name"`
		Address    string `json:"address"`
		ValidUntil int64  `json:"validUntil"`
	}
	if err := json.Unmarshal(raw, &agents); err != nil {
		// Some HL responses wrap differently; soft-empty.
		agents = nil
	}

	now := time.Now().UnixMilli()
	out := make([]gin.H, 0, len(agents))
	var primary gin.H
	for _, a := range agents {
		item := gin.H{
			"name":       a.Name,
			"address":    a.Address,
			"validUntil": a.ValidUntil,
		}
		out = append(out, item)
		if a.ValidUntil > now && primary == nil {
			primary = item
		}
	}

	builderApproved := false
	if feeRaw, feeErr := hlInfo(map[string]any{
		"type":    "maxBuilderFee",
		"user":    address,
		"builder": hyperliquidBuilder,
	}); feeErr == nil {
		var fee float64
		if json.Unmarshal(feeRaw, &fee) == nil && fee >= 50 {
			builderApproved = true
		} else {
			// Response may be a bare number without JSON number typing.
			var asInt int64
			if json.Unmarshal(feeRaw, &asInt) == nil && asInt >= 50 {
				builderApproved = true
			}
		}
	}

	writeJSON(c, http.StatusOK, gin.H{
		"agent":           primary,
		"agents":          out,
		"builderApproved": builderApproved,
	})
}

func (s *Service) consoleHyperliquidSubmitExchange(c *gin.Context) {
	var body struct {
		Action    map[string]any `json:"action"`
		Nonce     int64          `json:"nonce"`
		Signature map[string]any `json:"signature"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if body.Action == nil || body.Signature == nil {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "action and signature required"})
		return
	}

	payload := map[string]any{
		"action":    body.Action,
		"nonce":     body.Nonce,
		"signature": body.Signature,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(hyperliquidAPIURL+"/exchange", "application/json", bytes.NewReader(raw))
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		writeJSON(c, http.StatusBadGateway, gin.H{
			"success": false,
			"error":   fmt.Sprintf("hyperliquid exchange HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody))),
		})
		return
	}

	var parsed any
	_ = json.Unmarshal(respBody, &parsed)
	// Hyperliquid returns {"status":"ok",...} or {"status":"err","response":"..."}.
	if m, ok := parsed.(map[string]any); ok {
		if status, _ := m["status"].(string); strings.EqualFold(status, "err") {
			writeJSON(c, http.StatusBadRequest, gin.H{
				"success":  false,
				"error":    fmt.Sprintf("%v", m["response"]),
				"response": parsed,
			})
			return
		}
	}
	writeJSON(c, http.StatusOK, gin.H{
		"success":  true,
		"response": parsed,
	})
}
