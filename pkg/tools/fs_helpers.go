package tools

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// expandPath expands env vars and ~ to user home
func expandPath(p string) string {
	if p == "" {
		return p
	}
	p = os.ExpandEnv(p)
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			if len(p) >= 2 && (p[1] == '/' || p[1] == '\\') {
				return filepath.Join(home, p[2:])
			}
		}
	}
	return p
}

func splitRepoAndRef(s string) (string, string) {
	idx := strings.LastIndex(s, "#")
	if idx <= 0 || idx == len(s)-1 {
		return s, ""
	}
	return s[:idx], s[idx+1:]
}

// ensureVersionSuffix appends @latest when appropriate
func ensureVersionSuffix(spec string) string {
	if spec == "" || strings.Contains(spec, "@") {
		return spec
	}
	if filepath.IsAbs(spec) || strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../") ||
		strings.HasPrefix(spec, ".\\") || strings.HasPrefix(spec, "..\\") {
		return spec
	}
	if _, err := os.Stat(spec); err == nil {
		return spec
	}
	return spec + "@latest"
}

// envLookup find KEY=VAL entry
func envLookup(env []string, key string) string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return strings.TrimPrefix(e, prefix)
		}
	}
	return ""
}

func extractRepoName(u string) string {
	s := u
	if i := strings.LastIndex(s, "#"); i >= 0 {
		s = s[:i]
	}
	idx := strings.LastIndexAny(s, "/:")
	if idx >= 0 && idx+1 < len(s) {
		s = s[idx+1:]
	}
	s = strings.TrimSuffix(s, ".git")
	if s == "" {
		s = "repo"
	}
	return s
}

func sanitizeName(s string) string {
	r := strings.NewReplacer(
		"<", "-",
		">", "-",
		":", "-",
		"\"", "-",
		"|", "-",
		"?", "-",
		"*", "-",
	)
	s = r.Replace(s)
	s = strings.TrimSpace(s)
	if s == "" {
		return "repo"
	}
	return s
}

// copyFile copies file with 0644 permissions (overwrites target)
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	if mkErr := os.MkdirAll(filepath.Dir(dst), 0o755); mkErr != nil {
		return mkErr
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(0o644)
}

// isExecutable checks if filename is executable in dir (platform specific)
func isExecutable(name, dir string) bool {
	if runtime.GOOS == "windows" {
		lower := strings.ToLower(name)
		return strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".bat") || strings.HasSuffix(lower, ".cmd") || strings.HasSuffix(lower, ".ps1")
	}
	info, err := os.Stat(filepath.Join(dir, name))
	if err != nil {
		return false
	}
	mode := info.Mode()
	return mode&0o111 != 0
}

func stripExeSuffix(name string) string {
	if runtime.GOOS == "windows" {
		lower := strings.ToLower(name)
		for _, ext := range []string{".exe", ".bat", ".cmd", ".ps1"} {
			if strings.HasSuffix(lower, ext) {
				return name[:len(name)-len(ext)]
			}
		}
	}
	return name
}

// SnapshotExecutables quick snapshot of executable files modification time
func SnapshotExecutables(dir string) map[string]time.Time {
	m := make(map[string]time.Time)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return m
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isExecutable(name, dir) {
			continue
		}
		if fi, err := e.Info(); err == nil {
			m[name] = fi.ModTime()
		}
	}
	return m
}
