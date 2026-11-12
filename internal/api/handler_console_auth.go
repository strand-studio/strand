package api

import (
	"net/http"
	"strings"
	"time"

	"strand/internal/auth"
	"strand/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Service) consoleConfig(c *gin.Context) {
	st := s.eng.Store()
	initialized := false
	if st != nil {
		if n, err := st.User().Count(); err == nil {
			initialized = n > 0
		}
	}
	writeJSON(c, http.StatusOK, gin.H{
		"initialized": initialized,
		"beta_mode":   false,
	})
}

func (s *Service) consoleCryptoConfig(c *gin.Context) {
	writeJSON(c, http.StatusOK, gin.H{
		"transport_encryption": false,
	})
}

type authBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	BetaCode string `json:"beta_code"`
	Lang     string `json:"lang"`
}

func (s *Service) consoleRegister(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	var body authBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	email := strings.TrimSpace(strings.ToLower(body.Email))
	password := body.Password
	if email == "" || len(password) < 6 {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "邮箱或密码无效（密码至少 6 位）"})
		return
	}
	if existing, _ := st.User().GetByEmail(email); existing != nil {
		writeJSON(c, http.StatusConflict, gin.H{"error": "该邮箱已注册"})
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	user := &store.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := st.User().Create(user); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	_ = st.SetSystemConfig("initialized", "true")

	token, err := auth.GenerateJWT(user.ID, user.Email)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{
		"token":   token,
		"user_id": user.ID,
		"email":   user.Email,
		"message": "registered",
	})
}

func (s *Service) consoleLogin(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	var body authBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	email := strings.TrimSpace(strings.ToLower(body.Email))
	user, err := st.User().GetByEmail(email)
	if err != nil || user == nil {
		writeJSON(c, http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}
	if user.PasswordHash == "" || !auth.CheckPassword(body.Password, user.PasswordHash) {
		writeJSON(c, http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}
	token, err := auth.GenerateJWT(user.ID, user.Email)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{
		"token":   token,
		"user_id": user.ID,
		"email":   user.Email,
		"message": "ok",
	})
}

func (s *Service) consoleAdminLogin(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	_ = st.User().EnsureAdmin()
	admin, err := st.User().GetByID("admin")
	if err != nil || admin == nil {
		writeJSON(c, http.StatusUnauthorized, gin.H{"error": "管理员不可用"})
		return
	}
	// Prefer configured API token as admin password when set; otherwise accept any non-empty when hash empty (dev).
	ok := false
	if admin.PasswordHash != "" {
		ok = auth.CheckPassword(body.Password, admin.PasswordHash)
	} else if strings.TrimSpace(body.Password) != "" {
		ok = true
	}
	if !ok {
		writeJSON(c, http.StatusUnauthorized, gin.H{"error": "管理员密码错误"})
		return
	}
	token, err := auth.GenerateJWT(admin.ID, admin.Email)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{
		"token":   token,
		"user_id": admin.ID,
		"email":   admin.Email,
	})
}

func (s *Service) consoleLogout(c *gin.Context) {
	tok := extractToken(c.Request)
	if tok != "" {
		auth.BlacklistToken(tok, time.Now().Add(24*time.Hour))
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

func (s *Service) consoleChangePassword(c *gin.Context) {
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
	var body struct {
		NewPassword     string `json:"new_password"`
		CurrentPassword string `json:"current_password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	newPass := body.NewPassword
	if len(newPass) < 8 {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}
	user, err := st.User().GetByID(userID)
	if err != nil || user == nil {
		writeJSON(c, http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	if strings.TrimSpace(body.CurrentPassword) != "" && user.PasswordHash != "" {
		if !auth.CheckPassword(body.CurrentPassword, user.PasswordHash) {
			writeJSON(c, http.StatusBadRequest, gin.H{"error": "current password is incorrect"})
			return
		}
	}
	hash, err := auth.HashPassword(newPass)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	if err := st.User().UpdatePassword(userID, hash); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}
