package fanout

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"strand/internal/logger"
	"strand/internal/store"
)

// TelegramNotifier is the subset of the console Telegram binder we need.
type TelegramNotifier interface {
	Notify(text string) error
}

// Alerter fans runtime alerts to Telegram + Discord + Slack.
type Alerter struct {
	store    *store.Store
	telegram TelegramNotifier
	client   *http.Client
}

func New(st *store.Store, tg TelegramNotifier) *Alerter {
	return &Alerter{
		store:    st,
		telegram: tg,
		client:   &http.Client{Timeout: 12 * time.Second},
	}
}

// Notify delivers text to every configured console channel (best-effort).
func (a *Alerter) Notify(text string) {
	text = strings.TrimSpace(text)
	if text == "" || a == nil {
		return
	}
	if a.telegram != nil {
		if err := a.telegram.Notify(text); err != nil {
			logger.Debugf("telegram notify skipped: %v", err)
		}
	}
	if a.store == nil {
		return
	}
	for _, channelType := range []string{"discord", "slack"} {
		row, err := a.store.NotifyChannels().Get(channelType)
		if err != nil || row == nil {
			continue
		}
		switch channelType {
		case "discord":
			if err := a.notifyDiscord(row, text); err != nil {
				logger.Debugf("discord notify skipped: %v", err)
			}
		case "slack":
			if err := a.notifySlack(row, text); err != nil {
				logger.Debugf("slack notify skipped: %v", err)
			}
		}
	}
	// Unified messaging bindings (Web/CLI configured platforms).
	// Prefer per-user scene defaults (alert_platform_ids) when set; else all alerts_enabled.
	channels := a.messagingAlertTargets()
	for _, ch := range channels {
		if err := a.notifyMessagingChannel(ch, text); err != nil {
			logger.Debugf("messaging alert %s/%s skipped: %v", ch.PlatformType, ch.ID, err)
		}
	}
}

func (a *Alerter) messagingAlertTargets() []*store.MessagingChannel {
	if a == nil || a.store == nil {
		return nil
	}
	// Collect selected ids across users that have defaults; empty selection → all.
	defaults, err := a.store.Resources().ListDefaultsWithAlertSelection()
	if err != nil || len(defaults) == 0 {
		all, _ := a.store.Messaging().ListAllEnabledAlertChannels()
		return all
	}
	var selectedIDs []string
	anyRestriction := false
	for _, d := range defaults {
		ids := d.AlertIDs()
		if len(ids) == 0 {
			continue
		}
		anyRestriction = true
		selectedIDs = append(selectedIDs, ids...)
	}
	if !anyRestriction {
		all, _ := a.store.Messaging().ListAllEnabledAlertChannels()
		return all
	}
	rows, err := a.store.Messaging().ListEnabledAlertChannelsByIDs(selectedIDs)
	if err != nil {
		return nil
	}
	return rows
}

func (a *Alerter) notifyMessagingChannel(ch *store.MessagingChannel, text string) error {
	if ch == nil {
		return nil
	}
	creds, err := ch.CredentialMap()
	if err != nil {
		return err
	}
	switch strings.ToLower(ch.PlatformType) {
	case "telegram":
		token := strings.TrimSpace(creds["token"])
		peer := strings.TrimSpace(ch.BoundPeer)
		if peer == "" {
			peer = strings.TrimSpace(creds["chat_id"])
		}
		if token == "" || peer == "" {
			return fmt.Errorf("telegram alerts need token + bound_peer (chat id)")
		}
		return a.postJSON(
			fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token),
			map[string]any{"chat_id": peer, "text": truncate(text, 3500)},
		)
	case "discord":
		return a.notifyDiscord(&store.NotifyChannel{
			BotToken:    creds["token"],
			WebhookURL:  creds["webhook_url"],
			BoundChanID: firstNonEmpty(ch.BoundPeer, creds["channel_id"]),
		}, text)
	case "slack":
		return a.notifySlack(&store.NotifyChannel{
			BotToken:    creds["token"],
			WebhookURL:  creds["webhook_url"],
			BoundChanID: firstNonEmpty(ch.BoundPeer, creds["channel_id"]),
		}, text)
	default:
		// Chat platforms without a simple HTTP alert path still receive
		// conversation traffic; push alerts are best-effort for MVP.
		return fmt.Errorf("alert push not implemented for %s", ch.PlatformType)
	}
}

func (a *Alerter) notifyDiscord(row *store.NotifyChannel, text string) error {
	if webhook := strings.TrimSpace(row.WebhookURL); webhook != "" {
		return a.postJSON(webhook, map[string]any{
			"content": truncate(text, 1900),
		})
	}
	token := strings.TrimSpace(row.BotToken)
	channelID := strings.TrimSpace(row.BoundChanID)
	if token == "" || channelID == "" {
		return fmt.Errorf("discord needs webhook_url or bot_token+bound_channel_id")
	}
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", channelID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(mustJSON(map[string]any{
		"content": truncate(text, 1900),
	})))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+token)
	return a.do(req)
}

func (a *Alerter) notifySlack(row *store.NotifyChannel, text string) error {
	if webhook := strings.TrimSpace(row.WebhookURL); webhook != "" {
		return a.postJSON(webhook, map[string]any{
			"text": truncate(text, 3000),
		})
	}
	token := strings.TrimSpace(row.BotToken)
	channelID := strings.TrimSpace(row.BoundChanID)
	if token == "" || channelID == "" {
		return fmt.Errorf("slack needs webhook_url or bot_token+bound_channel_id")
	}
	req, err := http.NewRequest(http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewReader(mustJSON(map[string]any{
		"channel": channelID,
		"text":    truncate(text, 3000),
	})))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var parsed struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(body, &parsed)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack http %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	if !parsed.OK {
		return fmt.Errorf("slack api: %s", firstNonEmpty(parsed.Error, "failed"))
	}
	return nil
}

func (a *Alerter) postJSON(url string, payload map[string]any) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(mustJSON(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return a.do(req)
}

func (a *Alerter) do(req *http.Request) error {
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	return nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
