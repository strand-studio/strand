package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"strand/internal/mcp/payment"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
)

const (
	baseRPCURL          = "https://mainnet.base.org"
	hyperliquidBuilder  = "0x891dc6f05ad47a3c1a05da55e7a7517971faaf0d"
	hyperliquidBuilderFeeTenthsBps = "50" // 5 bps
)

func normalizePrivateKeyHex(raw string) (string, error) {
	key := strings.TrimSpace(raw)
	if key == "" {
		return "", fmt.Errorf("private_key required")
	}
	if strings.HasPrefix(strings.ToLower(key), "0x") {
		key = key[2:]
	}
	if len(key) != 64 {
		return "", fmt.Errorf("private key must be 32 bytes hex")
	}
	if _, err := hex.DecodeString(key); err != nil {
		return "", fmt.Errorf("invalid private key hex")
	}
	return "0x" + strings.ToLower(key), nil
}

func addressFromPrivateKey(privateKeyHex string) (string, error) {
	normalized, err := normalizePrivateKeyHex(privateKeyHex)
	if err != nil {
		return "", err
	}
	pk, err := crypto.HexToECDSA(strings.TrimPrefix(normalized, "0x"))
	if err != nil {
		return "", fmt.Errorf("invalid private key")
	}
	return crypto.PubkeyToAddress(pk.PublicKey).Hex(), nil
}

func (s *Service) consoleWalletGenerate(c *gin.Context) {
	pk, err := crypto.GenerateKey()
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	priv := hex.EncodeToString(crypto.FromECDSA(pk))
	addr := crypto.PubkeyToAddress(pk.PublicKey).Hex()
	writeJSON(c, http.StatusOK, gin.H{
		"address":     addr,
		"private_key": "0x" + priv,
	})
}

func (s *Service) consoleWalletValidate(c *gin.Context) {
	var body struct {
		PrivateKey string `json:"private_key"`
		DeriveOnly bool   `json:"derive_only"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	addr, err := addressFromPrivateKey(body.PrivateKey)
	if err != nil {
		writeJSON(c, http.StatusBadRequest, gin.H{"valid": false, "error": err.Error()})
		return
	}
	writeJSON(c, http.StatusOK, gin.H{
		"valid":   true,
		"address": addr,
	})
}

func (s *Service) consoleWalletBalance(c *gin.Context) {
	address := strings.TrimSpace(c.Query("address"))
	if !strings.HasPrefix(strings.ToLower(address), "0x") || len(address) != 42 {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "valid address required"})
		return
	}
	balance, err := fetchBaseUSDCBalance(address)
	if err != nil {
		// Soft-fail so the setup UI keeps polling instead of hard-erroring.
		writeJSON(c, http.StatusOK, gin.H{
			"address":      address,
			"balance_usdc": "0",
			"warning":      err.Error(),
		})
		return
	}
	writeJSON(c, http.StatusOK, gin.H{
		"address":      address,
		"balance_usdc": balance,
	})
}

func (s *Service) consoleWalletExport(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	var body struct {
		ModelID string `json:"model_id"`
		Address string `json:"address"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	userID := currentUserID(c)
	wantAddr := strings.ToLower(strings.TrimSpace(body.Address))

	tryModel := func(modelID string) (string, string, bool) {
		if modelID == "" {
			return "", "", false
		}
		m, err := st.AIModel().Get(userID, modelID)
		if err != nil || m == nil {
			return "", "", false
		}
		key := strings.TrimSpace(string(m.APIKey))
		if key == "" {
			return "", "", false
		}
		addr, err := addressFromPrivateKey(key)
		if err != nil {
			return "", "", false
		}
		if wantAddr != "" && strings.ToLower(addr) != wantAddr {
			return "", "", false
		}
		normalized, err := normalizePrivateKeyHex(key)
		if err != nil {
			return "", "", false
		}
		return addr, normalized, true
	}

	if addr, key, ok := tryModel(body.ModelID); ok {
		writeJSON(c, http.StatusOK, gin.H{"address": addr, "private_key": key})
		return
	}

	models, err := st.AIModel().List(userID)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	for _, m := range models {
		if m == nil || m.Provider != "claw402" {
			continue
		}
		if addr, key, ok := tryModel(m.ID); ok {
			writeJSON(c, http.StatusOK, gin.H{"address": addr, "private_key": key})
			return
		}
	}
	writeJSON(c, http.StatusNotFound, gin.H{"error": "wallet key not found for this account"})
}

func (s *Service) consoleHyperliquidConnectConfig(c *gin.Context) {
	writeJSON(c, http.StatusOK, gin.H{
		"builderAddress":   hyperliquidBuilder,
		"builderMaxFee":    hyperliquidBuilderFeeTenthsBps,
		"chain":            "Mainnet",
		"signatureChainId": "0xa4b1",
	})
}

func fetchBaseUSDCBalance(address string) (string, error) {
	addr := strings.TrimPrefix(strings.ToLower(address), "0x")
	if len(addr) != 40 {
		return "", fmt.Errorf("invalid address")
	}
	data := "0x70a08231" + strings.Repeat("0", 24) + addr
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_call",
		"params": []any{
			map[string]string{
				"to":   payment.BaseUSDCContract,
				"data": data,
			},
			"latest",
		},
	}
	raw, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Post(baseRPCURL, "application/json", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var parsed struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("%s", parsed.Error.Message)
	}
	hexBal := strings.TrimPrefix(strings.TrimSpace(parsed.Result), "0x")
	if hexBal == "" {
		return "0", nil
	}
	n := new(big.Int)
	if _, ok := n.SetString(hexBal, 16); !ok {
		return "", fmt.Errorf("invalid balance hex")
	}
	// USDC has 6 decimals.
	whole := new(big.Int).Div(n, big.NewInt(1_000_000))
	frac := new(big.Int).Mod(n, big.NewInt(1_000_000))
	return fmt.Sprintf("%s.%06d", whole.String(), frac.Int64()), nil
}
