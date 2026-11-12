package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var errStoreUnavailable = errors.New("database not ready")


func readJSON(r *http.Request, dest any) error {
	defer r.Body.Close()
	b, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return fmt.Errorf("请求体是空的")
	}
	if err := json.Unmarshal(b, dest); err != nil {
		return fmt.Errorf("JSON 解析失败: %w", err)
	}
	return nil
}

func writeJSON(c *gin.Context, status int, v any) {
	c.PureJSON(status, v)
}

func writeErr(c *gin.Context, status int, err error) {
	writeJSON(c, status, gin.H{"error": err.Error()})
}
