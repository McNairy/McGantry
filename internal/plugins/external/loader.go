package external

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScanDir returns the absolute paths of all executables in dir whose base name
// starts with "gantry-plugin-". Returns an empty slice (no error) when dir is
// empty or does not exist.
func ScanDir(dir string) ([]string, error) {
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan plugin dir %s: %w", dir, err)
	}

	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasPrefix(e.Name(), "gantry-plugin-") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Mode()&0o111 == 0 {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	return paths, nil
}

// PluginNameFromPath derives the plugin name from a binary path.
// e.g. "/opt/plugins/gantry-plugin-vcluster" -> "vcluster"
func PluginNameFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimPrefix(base, "gantry-plugin-")
}
