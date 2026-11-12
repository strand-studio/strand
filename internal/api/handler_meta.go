package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func agentsHandler(svc API) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := svc.ListAgents(c.Request.Context())
		if err != nil {
			writeErr(c, http.StatusInternalServerError, err)
			return
		}
		writeJSON(c, http.StatusOK, gin.H{"agents": body})
	}
}

func platformsHandler(svc API) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := svc.ListPlatforms(c.Request.Context())
		if err != nil {
			writeErr(c, http.StatusInternalServerError, err)
			return
		}
		writeJSON(c, http.StatusOK, gin.H{"platforms": body})
	}
}
