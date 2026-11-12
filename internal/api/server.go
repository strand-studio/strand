package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"strand/internal/engine"
	"strand/internal/logger"
	"strand/internal/settings"
)

// Serve starts the HTTP API and blocks until ctx is cancelled.
func Serve(ctx context.Context, cfg settings.Settings, eng *engine.Engine, version string) error {
	if !cfg.API.Enabled {
		logger.Info("API disabled in config")
		<-ctx.Done()
		return nil
	}
	if strings.TrimSpace(cfg.API.Token) == "" {
		logger.Warn("API token 为空：受保护接口将允许无鉴权访问（仅建议本机开发）")
	}

	svc := NewService(eng, version)
	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           Handler(svc, cfg),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("API 监听失败 (%s): %w", srv.Addr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Infof("API listening on http://%s", srv.Addr)
		errCh <- srv.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
