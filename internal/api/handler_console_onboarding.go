package api

import (
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
)

const beginnerDefaultModel = "deepseek"

type balanceCacheEntry struct {
	balance string
	status  string
	at      time.Time
}

var beginnerBalanceCache sync.Map // address(lower) → balanceCacheEntry

func cachedBaseUSDCBalance(address string) (balance string, status string) {
	key := strings.ToLower(strings.TrimSpace(address))
	if key == "" {
		return "0", "unknown"
	}
	if v, ok := beginnerBalanceCache.Load(key); ok {
		entry := v.(balanceCacheEntry)
		if time.Since(entry.at) < 30*time.Second {
			return entry.balance, entry.status
		}
	}
	bal, err := fetchBaseUSDCBalance(address)
	if err != nil {
		entry := balanceCacheEntry{balance: "0", status: "unknown", at: time.Now()}
		beginnerBalanceCache.Store(key, entry)
		return entry.balance, entry.status
	}
	entry := balanceCacheEntry{balance: bal, status: "ok", at: time.Now()}
	beginnerBalanceCache.Store(key, entry)
	return entry.balance, entry.status
}

func (s *Service) findClaw402Model(userID string) (modelID, address, privateKey string, enabled bool, ok bool) {
	st := s.eng.Store()
	if st == nil || userID == "" {
		return "", "", "", false, false
	}
	models, err := st.AIModel().List(userID)
	if err != nil {
		return "", "", "", false, false
	}
	// Prefer an enabled claw402 with a usable key.
	for _, m := range models {
		if m == nil || !strings.EqualFold(m.Provider, "claw402") {
			continue
		}
		key := strings.TrimSpace(string(m.APIKey))
		if key == "" {
			continue
		}
		addr, err := addressFromPrivateKey(key)
		if err != nil {
			continue
		}
		if m.Enabled {
			return m.ID, addr, key, true, true
		}
	}
	// Fall back to any claw402 with a key (we'll re-enable it).
	for _, m := range models {
		if m == nil || !strings.EqualFold(m.Provider, "claw402") {
			continue
		}
		key := strings.TrimSpace(string(m.APIKey))
		if key == "" {
			continue
		}
		addr, err := addressFromPrivateKey(key)
		if err != nil {
			continue
		}
		return m.ID, addr, key, m.Enabled, true
	}
	return "", "", "", false, false
}

func (s *Service) consoleBeginnerOnboarding(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	if userID == "" {
		writeJSON(c, http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	reused := false
	modelID, address, _, enabled, found := s.findClaw402Model(userID)
	if found {
		reused = true
		if !enabled {
			_ = st.AIModel().UpdateWithName(userID, modelID, "Claw402", true, "", "", beginnerDefaultModel)
		} else if modelID != "" {
			// Ensure beginner default model name is set when blank.
			if m, err := st.AIModel().Get(userID, modelID); err == nil && m != nil &&
				strings.TrimSpace(m.CustomModelName) == "" {
				_ = st.AIModel().UpdateWithName(userID, modelID, "", true, "", "", beginnerDefaultModel)
			}
		}
	} else {
		pk, err := crypto.GenerateKey()
		if err != nil {
			writeErr(c, http.StatusInternalServerError, err)
			return
		}
		priv := "0x" + hex.EncodeToString(crypto.FromECDSA(pk))
		address = crypto.PubkeyToAddress(pk.PublicKey).Hex()
		// Pass provider id "claw402" so UpdateWithName creates `{userID}_claw402`.
		if err := st.AIModel().UpdateWithName(
			userID,
			"claw402",
			"Claw402",
			true,
			priv,
			"",
			beginnerDefaultModel,
		); err != nil {
			writeErr(c, http.StatusInternalServerError, err)
			return
		}
		modelID, address, _, _, found = s.findClaw402Model(userID)
		if !found {
			writeJSON(c, http.StatusInternalServerError, gin.H{"error": "failed to persist beginner claw402 wallet"})
			return
		}
	}

	bal, status := cachedBaseUSDCBalance(address)
	writeJSON(c, http.StatusOK, gin.H{
		"address":             address,
		"chain":               "base",
		"asset":               "USDC",
		"provider":            "claw402",
		"default_model":       beginnerDefaultModel,
		"configured_model_id": modelID,
		"balance_usdc":        bal,
		"balance_status":      status,
		"env_saved":           true,
		"reused_existing":     reused,
		"ready":               true,
	})
}

func (s *Service) consoleBeginnerCurrent(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, gin.H{"found": false})
		return
	}
	userID := currentUserID(c)
	if userID == "" {
		writeJSON(c, http.StatusOK, gin.H{"found": false})
		return
	}
	modelID, address, _, _, found := s.findClaw402Model(userID)
	if !found {
		writeJSON(c, http.StatusOK, gin.H{"found": false})
		return
	}
	bal, status := cachedBaseUSDCBalance(address)
	writeJSON(c, http.StatusOK, gin.H{
		"found":           true,
		"address":         address,
		"balance_usdc":    bal,
		"balance_status":  status,
		"source":          "claw402",
		"claw402_status":  "ready",
		"configured_model_id": modelID,
	})
}
