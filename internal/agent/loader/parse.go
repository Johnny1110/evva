package loader

import (
	"fmt"
	"os"

	"github.com/johnny1110/evva/internal/tools"
	"gopkg.in/yaml.v3"
)

// toolsYml is the on-disk schema for <agent>/tools.yml.
type toolsYml struct {
	Active   []tools.ToolName `yaml:"active"`
	Deferred []tools.ToolName `yaml:"deferred"`
}

// metaYml is the on-disk schema for <agent>/meta.yml.
type metaYml struct {
	As        []string `yaml:"as"`
	Model     string   `yaml:"model"`
	WhenToUse string   `yaml:"when_to_use"`
}

func readToolsYml(path string) (toolsYml, error) {
	var out toolsYml
	b, err := os.ReadFile(path)
	if err != nil {
		return out, fmt.Errorf("read tools.yml: %w", err)
	}
	if err := yaml.Unmarshal(b, &out); err != nil {
		return out, fmt.Errorf("parse tools.yml: %w", err)
	}
	return out, nil
}

func readMetaYml(path string) (metaYml, error) {
	var out metaYml
	b, err := os.ReadFile(path)
	if err != nil {
		return out, fmt.Errorf("read meta.yml: %w", err)
	}
	if err := yaml.Unmarshal(b, &out); err != nil {
		return out, fmt.Errorf("parse meta.yml: %w", err)
	}
	return out, nil
}
