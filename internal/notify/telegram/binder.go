package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"strand/internal/logger"
	"strand/internal/store"
)

const apiBase = "https://api.telegram.org"

// Binder long-polls a console bot token, binds on /start, and can push alerts.
type Binder struct {
	mu     sync.Mutex
	store  *store.Store
	client *http.Client

	cancel context.CancelFunc
	token  string
	offset int64
}

func NewBinder(st *store.Store) *Binder {
	return &Binder{
		store:  st,
		client: &http.Client{Timeout: 45 * time.Second},
	}
}

// Start loads the saved token (if any) and begins polling.
func (b *Binder) Start(parent context.Context) {
	if b == nil || b.store == nil {
		return
	}
	cfg, err := b.store.TelegramConfig().Get()
	if err != nil || cfg == nil || strings.TrimSpace(cfg.BotToken) == "" {
		return
	}
	b.Restart(parent, cfg.BotToken)
}

// Restart replaces any running poller with one for token.
func (b *Binder) Restart(parent context.Context, token string) {
	if b == nil {
		return
	}
	token = strings.TrimSpace(token)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cancel != nil {
		b.cancel()
		b.cancel = nil
	}
	b.token = token
	b.offset = 0
	if token == "" {
		return
	}
	ctx, cancel := context.WithCancel(parent)
	b.cancel = cancel
	go b.pollLoop(ctx, token)
	logger.Infof("telegram binder polling started")
}

// Stop ends the poll loop.
func (b *Binder) Stop() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cancel != nil {
		b.cancel()
		b.cancel = nil
	}
	b.token = ""
}

// Notify sends a plain-text alert to the bound chat (no-op if unbound).
func (b *Binder) Notify(text string) error {
	if b == nil || b.store == nil {
		return fmt.Errorf("telegram binder unavailable")
	}
	cfg, err := b.store.TelegramConfig().Get()
	if err != nil || cfg == nil {
		return fmt.Errorf("telegram not configured")
	}
	if cfg.ChatID == 0 {
		return fmt.Errorf("telegram not bound — send /start to the bot first")
	}
	token := strings.TrimSpace(cfg.BotToken)
	if token == "" {
		return fmt.Errorf("telegram bot token missing")
	}
	return b.sendMessage(token, cfg.ChatID, text)
}

func (b *Binder) pollLoop(ctx context.Context, token string) {
	// Drop any pending webhook so getUpdates works for local/dev installs.
	_ = b.deleteWebhook(token)

	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		updates, err := b.getUpdates(ctx, token, b.offset)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Warnf("telegram binder getUpdates: %v", err)
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		for _, upd := range updates {
			if upd.UpdateID >= b.offset {
				b.offset = upd.UpdateID + 1
			}
			b.handleUpdate(token, upd)
		}
	}
}

type tgUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		MessageID int64 `json:"message_id"`
		Text      string `json:"text"`
		Chat      struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From *struct {
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"from"`
	} `json:"message"`
}

func (b *Binder) getUpdates(ctx context.Context, token string, offset int64) ([]tgUpdate, error) {
	url := fmt.Sprintf("%s/bot%s/getUpdates?timeout=25&offset=%d", apiBase, token, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	var parsed struct {
		OK          bool       `json:"ok"`
		Result      []tgUpdate `json:"result"`
		Description string     `json:"description"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if !parsed.OK {
		return nil, fmt.Errorf("%s", firstNonEmptyTG(parsed.Description, "getUpdates failed"))
	}
	return parsed.Result, nil
}

func (b *Binder) deleteWebhook(token string) error {
	url := fmt.Sprintf("%s/bot%s/deleteWebhook?drop_pending_updates=false", apiBase, token)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	return nil
}

func (b *Binder) handleUpdate(token string, upd tgUpdate) {
	if upd.Message == nil {
		return
	}
	text := strings.TrimSpace(upd.Message.Text)
	if text == "" {
		return
	}
	chatID := upd.Message.Chat.ID
	username := ""
	if upd.Message.From != nil {
		if u := strings.TrimSpace(upd.Message.From.Username); u != "" {
			username = "@" + u
		} else {
			username = strings.TrimSpace(upd.Message.From.FirstName + " " + upd.Message.From.LastName)
		}
	}

	fields := strings.Fields(text)
	if len(fields) == 0 {
		return
	}
	cmd, _, _ := strings.Cut(fields[0], "@")
	cmd = strings.ToLower(cmd)

	switch cmd {
	case "/start":
		if err := b.store.TelegramConfig().BindUser(chatID, username); err != nil {
			_ = b.sendMessage(token, chatID, "Bind failed: "+err.Error())
			return
		}
		display := username
		if display == "" {
			display = strconv.FormatInt(chatID, 10)
		}
		_ = b.sendMessage(token, chatID,
			"Bound to Strand ✅\n"+
				"Chat: "+display+"\n\n"+
				"You'll get Autopilot start/stop alerts here.\n"+
				"Commands: /status · /help · /unbind")
		logger.Infof("telegram bound chat_id=%d user=%s", chatID, username)

	case "/help":
		_ = b.sendMessage(token, chatID,
			"Strand console bot\n"+
				"/start — bind this chat for alerts\n"+
				"/status — running Autopilots\n"+
				"/unbind — stop alerts (keep bot token)\n"+
				"/help — this message")

	case "/status":
		_ = b.sendMessage(token, chatID, b.statusText())

	case "/unbind":
		boundID, _ := b.store.TelegramConfig().GetBoundChatID()
		if boundID != 0 && boundID != chatID {
			_ = b.sendMessage(token, chatID, "This chat is not the bound one.")
			return
		}
		if err := b.store.TelegramConfig().Unbind(); err != nil {
			_ = b.sendMessage(token, chatID, "Unbind failed: "+err.Error())
			return
		}
		_ = b.sendMessage(token, chatID, "Unbound. Send /start again anytime to re-bind.")

	default:
		// Ignore non-command chatter to avoid surprise AI costs.
		if strings.HasPrefix(cmd, "/") {
			_ = b.sendMessage(token, chatID, "Unknown command. Try /help")
		}
	}
}

func (b *Binder) statusText() string {
	if b.store == nil {
		return "Store unavailable"
	}
	traders, err := b.store.Trader().ListAll()
	if err != nil {
		return "Strand is online.\nCould not load Autopilots right now."
	}
	running := 0
	var lines []string
	for _, tr := range traders {
		if tr == nil || !tr.IsRunning {
			continue
		}
		running++
		name := strings.TrimSpace(tr.Name)
		if name == "" {
			name = tr.ID
		}
		lines = append(lines, "• "+name)
		if len(lines) >= 8 {
			break
		}
	}
	if running == 0 {
		return "Strand is online.\nNo Autopilots running.\nYou'll get start/stop alerts here."
	}
	out := fmt.Sprintf("Strand is online.\n%d Autopilot(s) running:\n%s", running, strings.Join(lines, "\n"))
	if running > len(lines) {
		out += fmt.Sprintf("\n…and %d more", running-len(lines))
	}
	return out
}

func (b *Binder) sendMessage(token string, chatID int64, text string) error {
	payload, _ := json.Marshal(map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "",
	})
	url := fmt.Sprintf("%s/bot%s/sendMessage", apiBase, token)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var parsed struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(body, &parsed)
	if resp.StatusCode >= 300 || !parsed.OK {
		return fmt.Errorf("sendMessage failed: %s", firstNonEmptyTG(parsed.Description, string(body)))
	}
	return nil
}

func firstNonEmptyTG(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
