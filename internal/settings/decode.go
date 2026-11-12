package settings

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

func decodeFile(path string, cfg *Settings) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}
	base := Default()
	*cfg = base
	if _, err := toml.Decode(string(data), cfg); err != nil {
		return fmt.Errorf("解析 TOML 失败 (%s): %w", path, err)
	}
	cfg.Path = path
	return nil
}
