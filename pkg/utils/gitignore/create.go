package gitignore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreateBaseGoGitignore creates a .gitignore file with the base Go template 通过 [tmplMap] key 获取模板
//
//  1. 需要检查目录是否存在
//  2. 需要检查是否已经存在 .gitignore 文件
//  3. 需要创建 .gitignore 文件并写入内容
func CreateBaseGoGitignore(dir string, template ...string) error {
	// ensure directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	gitignorePath := filepath.Join(dir, ".gitignore")

	// alias map to accept common synonyms
	alias := map[string]string{
		"goreleaser": "release",
		"go":         "base-go",
		"base":       "base-go",
	}

	// build the list of template keys to apply (preserve order, dedupe)
	wantKeys := make([]string, 0, len(template))
	seen := map[string]struct{}{}
	if len(template) == 0 {
		wantKeys = append(wantKeys, "base-go")
		seen["base-go"] = struct{}{}
	} else {
		for _, k := range template {
			if k == "" {
				continue
			}
			if a, ok := alias[k]; ok {
				k = a
			}
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			wantKeys = append(wantKeys, k)
		}
	}

	// prepare combined content for the requested templates
	var parts []string
	for _, k := range wantKeys {
		body, ok := tmplMap[k]
		if !ok {
			// fallback to base-go when unknown
			body = baseGoTmpl
		}
		header := fmt.Sprintf("# >>> gocli: template=%s >>>\n", k)
		footer := fmt.Sprintf("# <<< gocli: template=%s <<<\n", k)
		parts = append(parts, header+strings.TrimSpace(body)+"\n"+footer)
	}
	combined := strings.Join(parts, "\n") + "\n"

	// if file exists, read and append only missing template blocks
	if _, err := os.Stat(gitignorePath); err == nil {
		b, readErr := os.ReadFile(gitignorePath)
		if readErr != nil {
			return readErr
		}
		exist := string(b)
		// check for presence of each header; append those not present
		var toAppend []string
		for _, k := range wantKeys {
			header := fmt.Sprintf("# >>> gocli: template=%s >>>", k)
			if !strings.Contains(exist, header) {
				body, ok := tmplMap[k]
				if !ok {
					body = baseGoTmpl
				}
				footer := fmt.Sprintf("# <<< gocli: template=%s <<<\n", k)
				block := fmt.Sprintf("%s\n%s\n%s", header, strings.TrimSpace(body), footer)
				toAppend = append(toAppend, block)
			}
		}
		if len(toAppend) == 0 {
			// nothing to do
			return nil
		}
		// append to file (atomic: write to temp then rename)
		newContent := exist + "\n" + strings.Join(toAppend, "\n")
		tmpFile, readErr := os.CreateTemp(dir, ".gitignore.tmp.*")
		if readErr != nil {
			return readErr
		}
		tmpPath := tmpFile.Name()
		if _, wErr := tmpFile.WriteString(newContent); wErr != nil {
			tmpFile.Close()
			_ = os.Remove(tmpPath)
			return wErr
		}
		if closeErr := tmpFile.Close(); closeErr != nil {
			_ = os.Remove(tmpPath)
			return closeErr
		}
		if renameErr := os.Rename(tmpPath, gitignorePath); renameErr != nil {
			_ = os.Remove(tmpPath)
			return renameErr
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	// file does not exist: create with combined content
	tmpFile, err := os.CreateTemp(dir, ".gitignore.tmp.*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.WriteString(combined); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, gitignorePath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	// ensure permissions
	if err := os.Chmod(gitignorePath, 0o644); err != nil {
		return err
	}

	return nil
}
