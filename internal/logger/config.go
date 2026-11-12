package logger

type Config struct {
	Level string `json:"level"`
}

func (c *Config) SetDefault() {
	if c.Level == "" {
		c.Level = "info"
	}
}
