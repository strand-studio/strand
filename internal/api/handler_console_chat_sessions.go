package api

import (
	"net/http"
	"strings"

	"strand/internal/store"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (s *Service) consoleChatSessionsList(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	q := strings.TrimSpace(c.Query("q"))
	var (
		rows []store.ChatSession
		err  error
	)
	if q != "" {
		rows, err = st.ChatSessions().Search(currentUserID(c), q, 40)
	} else {
		rows, err = st.ChatSessions().List(currentUserID(c), 40)
	}
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"sessions": rows})
}

func (s *Service) consoleChatSessionCreate(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	var body struct {
		Title   string `json:"title"`
		ModelID string `json:"model_id"`
	}
	_ = c.ShouldBindJSON(&body)
	row, err := st.ChatSessions().Create(currentUserID(c), body.Title, body.ModelID)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, row)
}

func (s *Service) consoleChatSessionGet(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	row, err := st.ChatSessions().Get(currentUserID(c), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			writeJSON(c, http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	msgs, err := st.ChatSessions().ListMessages(currentUserID(c), id, 200)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"session": row, "messages": msgs})
}

func (s *Service) consoleChatSessionRename(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if err := st.ChatSessions().Rename(currentUserID(c), c.Param("id"), body.Title); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

func (s *Service) consoleChatSessionDelete(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	if err := st.ChatSessions().Delete(currentUserID(c), c.Param("id")); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}
