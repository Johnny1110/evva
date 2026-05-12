package config

import (
	"os"
)

// setupGlobalParam ensures the global config directories exist.
func setupGlobalParam(cfg *AppConfig) {
	_ = os.MkdirAll(cfg.GlobalCfgDir, 0o755)
	_ = os.MkdirAll(cfg.GlobalSkillsDir, 0o755)
}
