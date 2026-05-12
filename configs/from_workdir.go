package config

import "os"

// setupWorkDirParam sets WorkDir to the process's current working directory
// and derives WorkDirSkillsDir from it.
func setupWorkDirParam(cfg *AppConfig) {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	cfg.WorkDir = wd
	cfg.WorkDirSkillsDir = wd + "/skills"
}
