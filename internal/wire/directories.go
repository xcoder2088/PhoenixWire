package wire

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Resolve existing ancestors too, so a not-yet-created child of a symlink
// cannot redirect a Wire key/vault into PhoenixMail's independent claim service.
func resolvedDirectory(path string) (string, error) {
	return resolveDirectory(path, 0)
}
func resolveDirectory(path string, depth int) (string, error) {
	if depth > 40 {
		return "", errors.New("directory_symlink_loop")
	}
	if !filepath.IsAbs(path) {
		return "", errors.New("absolute_wire_directory_required")
	}
	path = filepath.Clean(path)
	probe := path
	var tail []string
	for {
		resolved, err := filepath.EvalSymlinks(probe)
		if err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if info, e := os.Lstat(probe); e == nil && info.Mode()&os.ModeSymlink != 0 {
			target, e := os.Readlink(probe)
			if e != nil {
				return "", e
			}
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(probe), target)
			}
			for i := len(tail) - 1; i >= 0; i-- {
				target = filepath.Join(target, tail[i])
			}
			return resolveDirectory(target, depth+1)
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", err
		}
		tail = append(tail, filepath.Base(probe))
		probe = parent
	}
}
func containsDirectory(parent, child string) bool {
	return parent == child || strings.HasPrefix(child, parent+string(os.PathSeparator))
}
func separateWireDirectories(config, state string) error {
	for _, path := range []string{config, state} {
		resolved, err := resolvedDirectory(path)
		if err != nil {
			return err
		}
		for _, reserved := range []string{"/etc/nimiqsecureclaim", "/var/lib/nimiqsecureclaim"} {
			target, err := resolvedDirectory(reserved)
			if err != nil {
				return err
			}
			if containsDirectory(target, resolved) || containsDirectory(resolved, target) {
				return errors.New("wire_requires_independent_directories")
			}
		}
	}
	return nil
}
