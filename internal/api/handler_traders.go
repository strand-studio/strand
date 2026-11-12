package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func tradersListHandler(svc API) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := svc.ListTraders(c.Request.Context())
		if err != nil {
			writeErr(c, http.StatusInternalServerError, err)
			return
		}
		writeJSON(c, http.StatusOK, gin.H{"traders": body})
	}
}

func traderGetHandler(svc API) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := svc.GetTrader(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, http.StatusNotFound, err)
			return
		}
		writeJSON(c, http.StatusOK, body)
	}
}

func traderStartHandler(svc API) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.StartTrader(c.Request.Context(), c.Param("id")); err != nil {
			writeErr(c, http.StatusBadRequest, err)
			return
		}
		writeJSON(c, http.StatusOK, gin.H{"ok": true})
	}
}

func traderStopHandler(svc API) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.StopTrader(c.Request.Context(), c.Param("id")); err != nil {
			writeErr(c, http.StatusBadRequest, err)
			return
		}
		writeJSON(c, http.StatusOK, gin.H{"ok": true})
	}
}
