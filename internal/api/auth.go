package api

import (
	"net/http"
	"strings"

	"strand/internal/auth"

	"github.com/gin-gonic/gin"
)

func requireAuth(apiToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authorized(apiToken, c.Request) {
			c.Next()
			return
		}
		writeJSON(c, http.StatusUnauthorized, gin.H{"error": "未授权：请提供 X-Strand-Token 或 Bearer token"})
		c.Abort()
	}
}

func authorized(apiToken string, r *http.Request) bool {
	tok := extractToken(r)
	if tok == "" {
		// No token configured → allow only loopback is too surprising; require token if set.
		return strings.TrimSpace(apiToken) == ""
	}

	if strings.TrimSpace(apiToken) != "" && tok == apiToken {
		return true
	}

	// Also accept user JWT issued by auth package.
	if claims, err := auth.ValidateJWT(tok); err == nil && claims != nil {
		if !auth.IsTokenBlacklisted(tok) {
			return true
		}
	}
	return false
}

func extractToken(r *http.Request) string {
	tok := strings.TrimSpace(r.Header.Get("X-Strand-Token"))
	if tok != "" {
		return tok
	}
	h := r.Header.Get("Authorization")
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
}
