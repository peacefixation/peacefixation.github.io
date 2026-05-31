package site

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

)

// stemOf returns the filename stem of path (base name without extension).
func stemOf(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// isDraft reports whether item data contains draft: true.
func isDraft(data map[string]any) bool {
	b, ok := data["draft"].(bool)
	return ok && b
}

// first returns the first non-empty string from the arguments.
func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// copyStaticDir copies all files from src into outputDir/static/, preserving
// directory structure. Skips silently if src does not exist.
func copyStaticDir(src, outputDir string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	destRoot := filepath.Join(outputDir, "static")
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(destRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		return copyFile(path, dest)
	})
}

// copyFiles copies all files whose extension is in exts from src into dest,
// preserving filenames. Skips silently if src does not exist.
func copyFiles(src, dest string, exts map[string]bool) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !exts[ext] {
			continue
		}
		if err := copyFile(filepath.Join(src, entry.Name()), filepath.Join(dest, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

