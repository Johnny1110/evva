// Package banner exposes the evva splash banner. The default banner is
// embedded into the binary; callers may override it by dropping their
// own banner.txt into the user's EvvaHome (typically ~/.evva/).
package banner

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed banner.txt
var defaultBanner string

// Default returns the banner shipped with the binary.
func Default() string { return defaultBanner }

// Load returns the banner the user wants displayed. It looks for
// `<evvaHome>/banner.txt` first and falls back to the embedded default
// when that file is missing, empty, or unreadable.
//
// Empty evvaHome skips the disk lookup entirely. Errors during the
// lookup are swallowed — banners are decorative, never load-bearing.
func Load(evvaHome string) string {
	if evvaHome != "" {
		path := filepath.Join(evvaHome, "banner.txt")
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			return string(data)
		}
	}
	return defaultBanner
}
