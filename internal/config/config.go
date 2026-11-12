package config

import (
	"fmt"
	"os"
	"strings"

	"strand/internal/auth"
	"strand/internal/mcp"
	"strand/internal/settings"
	"strand/internal/telemetry"
)

const (
	insecureDefaultJWTSecret = "default-jwt-secret-change-in-production"
	minJWTSecretLength       = 32
)

// ConfigPath stores the path to the loaded TOML (for display / save).
var ConfigPath string

var global *Config

// Config is the process-wide runtime view used by providers and helpers.
type Config struct {
	Settings settings.Settings

	JWTSecret string

	APISeverPort int // kept spelling for compatibility with older call sites
	APIHost      string
	APIToken     string

	DBType     string
	DBPath     string
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	ExperienceImprovement bool

	AlpacaAPIKey    string
	AlpacaSecretKey string
	TwelveDataKey   string
}

// Apply builds the global runtime config from settings and JWT env.
func Apply(s settings.Settings) error {
	cfg := &Config{
		Settings:              s,
		APISeverPort:          s.API.Port,
		APIHost:               s.API.Host,
		APIToken:              s.API.Token,
		DBType:                s.Database.Type,
		DBPath:                s.Database.Path,
		DBHost:                s.Database.Host,
		DBPort:                s.Database.Port,
		DBUser:                s.Database.User,
		DBPassword:            s.Database.Password,
		DBName:                s.Database.Name,
		DBSSLMode:             s.Database.SSLMode,
		ExperienceImprovement: s.Runtime.ExperienceImprovement,
		AlpacaAPIKey:          s.Market.AlpacaAPIKey,
		AlpacaSecretKey:       s.Market.AlpacaSecretKey,
		TwelveDataKey:         s.Market.TwelveDataKey,
	}

	jwt := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwt == "" {
		return fmt.Errorf("JWT_SECRET 未设置：请在 %s/.env 中配置，或先运行 strand config init", s.Data.Dir)
	}
	if jwt == insecureDefaultJWTSecret {
		return fmt.Errorf("JWT_SECRET 仍是不安全默认值，请重新生成（openssl rand -base64 48）")
	}
	if len(jwt) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET 至少需要 %d 字节（当前 %d）", minJWTSecretLength, len(jwt))
	}
	cfg.JWTSecret = jwt

	auth.SetJWTSecret(jwt)
	telemetry.Init(cfg.ExperienceImprovement, "")

	mcp.TokenUsageCallback = func(usage mcp.TokenUsage) {
		telemetry.TrackAIUsage(telemetry.AIUsageEvent{
			ModelProvider: usage.Provider,
			ModelName:     usage.Model,
			Channel:       usage.Channel(),
			InputTokens:   usage.PromptTokens,
			OutputTokens:  usage.CompletionTokens,
		})
	}

	ConfigPath = s.Path
	global = cfg
	return nil
}

// Init is a no-op placeholder kept for older call sites; prefer Apply.
func Init() {}

// Get returns the global configuration (may be nil before Apply).
func Get() *Config {
	if global == nil {
		return &Config{}
	}
	return global
}

// MustGet returns the global config or panics if Apply was not called.
func MustGet() *Config {
	if global == nil {
		panic("config: Apply() was not called")
	}
	return global
}
