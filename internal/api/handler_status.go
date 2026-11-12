package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func statusHandler(svc API) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := svc.Status(c.Request.Context())
		if err != nil {
			writeErr(c, http.StatusInternalServerError, err)
			return
		}
		writeJSON(c, http.StatusOK, body)
	}
}
