package settings

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var envPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func expandSettings(cfg *Settings) {
	cfg.Data.Dir = expandString(cfg.Data.Dir)
	cfg.API.Host = expandString(cfg.API.Host)
	cfg.API.Token = expandString(cfg.API.Token)
	cfg.API.Mode = expandString(cfg.API.Mode)
	cfg.Database.Path = expandString(cfg.Database.Path)
	cfg.Database.Host = expandString(cfg.Database.Host)
	cfg.Database.User = expandString(cfg.Database.User)
	cfg.Database.Password = expandString(cfg.Database.Password)
	cfg.Database.Name = expandString(cfg.Database.Name)
	cfg.Database.SSLMode = expandString(cfg.Database.SSLMode)
	cfg.Runtime.Language = expandString(cfg.Runtime.Language)
	cfg.Runtime.LogLevel = expandString(cfg.Runtime.LogLevel)
	cfg.Market.AlpacaAPIKey = expandString(cfg.Market.AlpacaAPIKey)
	cfg.Market.AlpacaSecretKey = expandString(cfg.Market.AlpacaSecretKey)
	cfg.Market.TwelveDataKey = expandString(cfg.Market.TwelveDataKey)

	for i := range cfg.Projects {
		p := &cfg.Projects[i]
		p.Name = expandString(p.Name)
		p.WorkDir = expandString(p.WorkDir)
		p.Language = expandString(p.Language)
		p.Agent.Type = expandString(p.Agent.Type)
		p.Agent.Model = expandString(p.Agent.Model)
		p.Agent.Options = expandMap(p.Agent.Options)
		for j := range p.Platforms {
			p.Platforms[j] = expandMap(p.Platforms[j])
		}
	}
}

func expandString(s string) string {
	if s == "" || !strings.Contains(s, "${") {
		return s
	}
	return envPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := envPattern.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		if v, ok := os.LookupEnv(sub[1]); ok {
			return v
		}
		return ""
	})
}

func expandMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch t := v.(type) {
		case string:
			out[k] = expandString(t)
		case map[string]any:
			out[k] = expandMap(t)
		default:
			out[k] = v
		}
	}
	return out
}

func applyEnvOverrides(cfg *Settings) {
	if v := strings.TrimSpace(os.Getenv("STRAND_API_TOKEN")); v != "" {
		cfg.API.Token = v
	}
	if v := strings.TrimSpace(os.Getenv("API_TOKEN")); v != "" && cfg.API.Token == "" {
		cfg.API.Token = v
	}
	if v := strings.TrimSpace(os.Getenv("API_SERVER_PORT")); v != "" {
		if p := atoi(v); p > 0 {
			cfg.API.Port = p
		}
	}
	if v := strings.TrimSpace(os.Getenv("STRAND_API_PORT")); v != "" {
		if p := atoi(v); p > 0 {
			cfg.API.Port = p
		}
	}
	if v := strings.TrimSpace(os.Getenv("STRAND_API_HOST")); v != "" {
		cfg.API.Host = v
	}
	if v := strings.TrimSpace(os.Getenv("GIN_MODE")); v != "" {
		cfg.API.Mode = v
	}
	if v := strings.TrimSpace(os.Getenv("STRAND_GIN_MODE")); v != "" {
		cfg.API.Mode = v
	}
	if v := strings.TrimSpace(os.Getenv("DB_TYPE")); v != "" {
		cfg.Database.Type = strings.ToLower(v)
	}
	if v := strings.TrimSpace(os.Getenv("DB_PATH")); v != "" {
		cfg.Database.Path = v
	}
	if v := strings.TrimSpace(os.Getenv("DB_HOST")); v != "" {
		cfg.Database.Host = v
	}
	if v := strings.TrimSpace(os.Getenv("DB_PORT")); v != "" {
		if p := atoi(v); p > 0 {
			cfg.Database.Port = p
		}
	}
	if v := strings.TrimSpace(os.Getenv("DB_USER")); v != "" {
		cfg.Database.User = v
	}
	if v := strings.TrimSpace(os.Getenv("DB_PASSWORD")); v != "" {
		cfg.Database.Password = v
	}
	if v := strings.TrimSpace(os.Getenv("DB_NAME")); v != "" {
		cfg.Database.Name = v
	}
	if v := strings.TrimSpace(os.Getenv("DB_SSLMODE")); v != "" {
		cfg.Database.SSLMode = v
	}
	if v := strings.TrimSpace(os.Getenv("EXPERIENCE_IMPROVEMENT")); v != "" {
		cfg.Runtime.ExperienceImprovement = strings.ToLower(v) != "false"
	}
	if v := strings.TrimSpace(os.Getenv("ALPACA_API_KEY")); v != "" {
		cfg.Market.AlpacaAPIKey = v
	}
	if v := strings.TrimSpace(os.Getenv("ALPACA_SECRET_KEY")); v != "" {
		cfg.Market.AlpacaSecretKey = v
	}
	if v := strings.TrimSpace(os.Getenv("TWELVE_DATA_KEY")); v != "" {
		cfg.Market.TwelveDataKey = v
	}
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func loadDotEnvCandidates(configPath string) {
	seen := map[string]bool{}
	try := func(p string) {
		p = filepath.Clean(p)
		if p == "" || p == "." || seen[p] {
			return
		}
		seen[p] = true
		_ = loadDotEnvFile(p)
	}

	try(".env")
	if configPath != "" {
		try(filepath.Join(filepath.Dir(configPath), ".env"))
	}
	try(filepath.Join(DefaultDataDir(), ".env"))
}

// loadDotEnvFile loads KEY=VALUE lines without overriding existing env.
func loadDotEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, val)
	}
	return sc.Err()
}
