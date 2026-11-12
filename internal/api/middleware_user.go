package api

import (
	"net/http"
	"strings"

	"strand/internal/auth"

	"github.com/gin-gonic/gin"
)

const (
	ctxUserIDKey = "strandUserID"
	ctxEmailKey  = "strandEmail"
)

// requireUserAuth accepts a Strand API token (treated as operator) or a user JWT.
func requireUserAuth(apiToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := extractToken(c.Request)
		if tok == "" {
			if strings.TrimSpace(apiToken) == "" {
				// Dev mode: anonymous operator.
				c.Set(ctxUserIDKey, "admin")
				c.Set(ctxEmailKey, "admin@localhost")
				c.Next()
				return
			}
			writeJSON(c, http.StatusUnauthorized, gin.H{"error": "未授权：请先登录"})
			c.Abort()
			return
		}

		if strings.TrimSpace(apiToken) != "" && tok == apiToken {
			c.Set(ctxUserIDKey, "admin")
			c.Set(ctxEmailKey, "admin@localhost")
			c.Next()
			return
		}

		claims, err := auth.ValidateJWT(tok)
		if err != nil || claims == nil || auth.IsTokenBlacklisted(tok) {
			writeJSON(c, http.StatusUnauthorized, gin.H{"error": "会话已过期，请重新登录"})
			c.Abort()
			return
		}
		c.Set(ctxUserIDKey, claims.UserID)
		c.Set(ctxEmailKey, claims.Email)
		c.Next()
	}
}

func currentUserID(c *gin.Context) string {
	if v, ok := c.Get(ctxUserIDKey); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return ""
}
