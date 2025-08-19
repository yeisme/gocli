package tools

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func expandPath(p string) string {
	if p == "" {
		return p
	}
	// 展开环境变量
	p = os.ExpandEnv(p)
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			// 处理 ~/xxx
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

// ensureVersionSuffix 在满足以下条件时为 spec 追加 @latest：
//   - 没有包含 '@'
//   - 不是本地路径（不是绝对路径；不以 ./ 或 ../ 开头；路径在本地不存在）
func ensureVersionSuffix(spec string) string {
	if spec == "" || strings.Contains(spec, "@") {
		return spec
	}

	// 明显的本地路径：绝对路径、./ 或 ../
	if filepath.IsAbs(spec) || strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../") ||
		strings.HasPrefix(spec, ".\\") || strings.HasPrefix(spec, "..\\") {
		return spec
	}

	// 若文件系统上确实存在该路径（目录或文件），视为本地
	if _, err := os.Stat(spec); err == nil {
		return spec
	}

	// 其他情况，当作模块路径，追加 @latest
	return spec + "@latest"
}

// envLookup 在形如 KEY=VAL 的切片中查找 KEY 的值
func envLookup(env []string, key string) string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return strings.TrimPrefix(e, prefix)
		}
	}
	return ""
}

// extractRepoName 尽力从仓库 URL 中提取仓库名（去除 .git 尾缀）
func extractRepoName(u string) string {
	s := u
	// 去掉片段
	if i := strings.LastIndex(s, "#"); i >= 0 {
		s = s[:i]
	}
	// 兼容 ssh/https：取最后一个路径分隔符后的名字
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

// sanitizeName 清理为文件夹安全名（Windows 也安全）
func sanitizeName(s string) string {
	// 替换常见非法字符
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

// copyFile 以 0644 权限复制普通文件（若目标存在则覆盖）
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	// 确保目标目录存在
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
