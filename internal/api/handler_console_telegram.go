package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func maskTelegramToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 || len(parts[1]) < 6 {
		if len(token) <= 8 {
			return "***"
		}
		return token[:4] + "***" + token[len(token)-3:]
	}
	secret := parts[1]
	if len(secret) <= 6 {
		return parts[0] + ":***"
	}
	return parts[0] + ":" + secret[:3] + "***" + secret[len(secret)-3:]
}

func (s *Service) consoleTelegramGet(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, gin.H{"token_masked": "", "is_bound": false})
		return
	}
	cfg, err := st.TelegramConfig().Get()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(c, http.StatusOK, gin.H{"token_masked": "", "is_bound": false})
			return
		}
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{
		"token_masked":  maskTelegramToken(cfg.BotToken),
		"is_bound":      cfg.ChatID != 0,
		"bound_chat_id": cfg.ChatID,
		"username":      cfg.Username,
		"model_id":      cfg.ModelID,
	})
}

func (s *Service) consoleTelegramSave(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	var body struct {
		BotToken string `json:"bot_token"`
		ModelID  string `json:"model_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	token := strings.TrimSpace(body.BotToken)
	if token == "" || !strings.Contains(token, ":") {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "valid Telegram bot token required (from @BotFather)"})
		return
	}
	if err := st.TelegramConfig().Save(token, strings.TrimSpace(body.ModelID)); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	s.eng.RestartTelegramBinder(token)
	writeJSON(c, http.StatusOK, gin.H{
		"ok":           true,
		"token_masked": maskTelegramToken(token),
		"is_bound":     false,
		"hint":         "Open Telegram, message your bot, then send /start to bind this Strand instance.",
	})
}

func (s *Service) consoleTelegramUnbind(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	if err := st.TelegramConfig().Unbind(); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true, "is_bound": false})
}

func (s *Service) consoleTelegramSetModel(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	var body struct {
		ModelID string `json:"model_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	cfg, err := st.TelegramConfig().Get()
	if err != nil {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "save a Telegram bot token first"})
		return
	}
	if err := st.TelegramConfig().Save(cfg.BotToken, strings.TrimSpace(body.ModelID)); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true, "model_id": body.ModelID})
}
