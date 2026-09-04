package tonalt1arms

import (
	"os"
	"path/filepath"
)

// RepoPathHelper resolves a path relative to this module's root (the
// directory containing go.mod), so frozen artifact paths under
// experiments/tonal-t1/ resolve correctly whether the caller is `go test`
// (cwd = package directory) or a built binary invoked from the module root.
// Mirrors internal/tonalt1.RepoPath; duplicated rather than imported to
// avoid coupling two sibling packages that are otherwise independent.
func RepoPathHelper(relative string) string {
	return filepath.Join(moduleRootHelper(), relative)
}

func moduleRootHelper() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}
