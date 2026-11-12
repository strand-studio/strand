package settings

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InitOptions controls config init.
type InitOptions struct {
	Force      bool
	ConfigPath string // empty → DefaultConfigPath()
	DataDir    string // empty → DefaultDataDir()
}

// InitResult describes what was written.
type InitResult struct {
	ConfigPath string
	EnvPath    string
	DataDir    string
	Created    []string
}

// WriteInit creates data dir, config.toml and .env (secrets).
func WriteInit(opt InitOptions) (InitResult, error) {
	var out InitResult

	dataDir := ExpandHome(opt.DataDir)
	if dataDir == "" {
		dataDir = DefaultDataDir()
	}
	out.DataDir = dataDir

	cfgPath := ExpandHome(opt.ConfigPath)
	if cfgPath == "" {
		cfgPath = filepath.Join(dataDir, "config.toml")
	}
	out.ConfigPath = cfgPath
	out.EnvPath = filepath.Join(dataDir, ".env")

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return out, fmt.Errorf("创建数据目录失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "workspace"), 0o755); err != nil {
		return out, err
	}

	if !opt.Force {
		if _, err := os.Stat(cfgPath); err == nil {
			return out, fmt.Errorf("配置已存在: %s（使用 --force 覆盖）", cfgPath)
		}
	}

	jwt, err := randomSecret(48)
	if err != nil {
		return out, err
	}
	apiToken, err := randomSecret(32)
	if err != nil {
		return out, err
	}

	tomlBody := strings.ReplaceAll(ExampleTOML, "{{DATA_DIR}}", filepath.ToSlash(dataDir))
	if err := os.WriteFile(cfgPath, []byte(tomlBody), 0o644); err != nil {
		return out, fmt.Errorf("写入配置失败: %w", err)
	}
	out.Created = append(out.Created, cfgPath)

	envBody := fmt.Sprintf(`# Strand secrets — do not commit
JWT_SECRET=%s
STRAND_API_TOKEN=%s

# Optional market providers
# ALPACA_API_KEY=
# ALPACA_SECRET_KEY=
# TWELVE_DATA_KEY=

# Messaging platform tokens (referenced from config.toml via ${...})
# WEIXIN_TOKEN=
# TELEGRAM_BOT_TOKEN=
`, jwt, apiToken)

	if !opt.Force {
		if _, err := os.Stat(out.EnvPath); err == nil {
			// keep existing .env
			return out, nil
		}
	}
	if err := os.WriteFile(out.EnvPath, []byte(envBody), 0o600); err != nil {
		return out, fmt.Errorf("写入 .env 失败: %w", err)
	}
	out.Created = append(out.Created, out.EnvPath)
	return out, nil
}

func randomSecret(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ExampleTOML is the starter config written by `strand config init`.
const ExampleTOML = `# Strand 主配置（结构放这里；密钥放同目录 .env）
#
# 优先级: CLI flag > 环境变量 / .env > 本文件 > 默认值

[data]
dir = "{{DATA_DIR}}"

[api]
enabled = true
host = "127.0.0.1"
port = 8080
mode = "release"         # debug | release | test（也可用 GIN_MODE / STRAND_GIN_MODE）
# 也可用环境变量 STRAND_API_TOKEN
token = "${STRAND_API_TOKEN}"

[database]
type = "sqlite"          # sqlite | postgres
path = "data.db"         # 相对 data.dir；postgres 时忽略
# host = "localhost"
# port = 5432
# user = "postgres"
# password = "${DB_PASSWORD}"
# name = "strand"
# sslmode = "disable"

[runtime]
language = "zh"
log_level = "info"
experience_improvement = true
auto_start_traders = true

[trading]
enabled = true

[market]
# alpaca_api_key = "${ALPACA_API_KEY}"
# alpaca_secret_key = "${ALPACA_SECRET_KEY}"
# twelve_data_key = "${TWELVE_DATA_KEY}"

# —— 消息通道（可选；不配则仅 API / 自动交易）——
# [[projects]]
# name = "default"
# work_dir = "{{DATA_DIR}}/workspace/default"
# language = "zh"
#
#   [projects.agent]
#   type = "codex"          # codex | gemini | claudecode | copilot | cursor | kimi | opencode
#   model = ""
#
#     [projects.agent.options]
#     # work_dir 等透传给 Agent
#
#   [[projects.platforms]]
#   type = "weixin"
#   token = "${WEIXIN_TOKEN}"
#   allow_from = ""
`
