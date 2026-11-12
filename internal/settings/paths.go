package settings

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultDataDir is ~/.strand (or %USERPROFILE%\.strand on Windows).
func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".strand"
	}
	return filepath.Join(home, ".strand")
}

// DefaultConfigPath is ~/.strand/config.toml.
func DefaultConfigPath() string {
	return filepath.Join(DefaultDataDir(), "config.toml")
}

// ResolveConfigPath picks the TOML file to load.
// Order: explicit path → STRAND_CONFIG → ./config.toml → ~/.strand/config.toml.
// Returns "" if none exist (caller may use defaults).
func ResolveConfigPath(explicit string) (string, error) {
	candidates := make([]string, 0, 4)
	if strings.TrimSpace(explicit) != "" {
		candidates = append(candidates, ExpandHome(explicit))
	}
	if v := strings.TrimSpace(os.Getenv("STRAND_CONFIG")); v != "" {
		candidates = append(candidates, ExpandHome(v))
	}
	candidates = append(candidates,
		"config.toml",
		DefaultConfigPath(),
	)

	seen := map[string]bool{}
	for _, c := range candidates {
		c = filepath.Clean(c)
		if c == "." || c == "" || seen[c] {
			continue
		}
		seen[c] = true
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			abs, err := filepath.Abs(c)
			if err != nil {
				return c, nil
			}
			return abs, nil
		}
	}
	// Prefer default path even if missing (for init / error messages).
	if strings.TrimSpace(explicit) != "" {
		return ExpandHome(explicit), nil
	}
	return "", nil
}

// ExpandHome replaces a leading ~/ with the user home directory.
func ExpandHome(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
