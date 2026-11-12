package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func healthHandler(svc API) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := svc.Health(c.Request.Context())
		if err != nil {
			writeErr(c, http.StatusInternalServerError, err)
			return
		}
		writeJSON(c, http.StatusOK, body)
	}
}
