package newproject

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CopyTemplateFSToDir 将提供的 fs.FS(模板) 递归复制到 destDir
// 参数:
//
//	src: 模板文件系统（如 embed / zip 解压目录等）
//	destDir: 目标项目根目录（必须存在或可创建）
//
// 规则:
//   - 保留相对路径结构
//   - 若目标文件已存在则覆盖
//   - 跳过 .git/ 目录与其内容
//   - 跳过空目录仅在需要时创建
//   - 符号链接/设备文件直接报错并中止
func CopyTemplateFSToDir(src fs.FS, destDir string, force bool) error {
	if src == nil {
		return fmt.Errorf("source fs is nil")
	}
	if strings.TrimSpace(destDir) == "" {
		destDir = "."
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	var errs []string
	walkErr := fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, fmt.Sprintf("walk %s: %v", path, err))
			return nil // 继续收集其它错误
		}
		// 跳过 .git 目录
		if path == ".git" || strings.HasPrefix(path, ".git/") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path == "." { // 根目录本身
			return nil
		}

		target := filepath.Join(destDir, path)
		if d.IsDir() {
			if err = os.MkdirAll(target, 0o755); err != nil {
				errs = append(errs, fmt.Sprintf("mkdir %s: %v", target, err))
			}
			return nil
		}

		// 非常规文件类型处理
		if !d.Type().IsRegular() {
			errs = append(errs, fmt.Sprintf("unsupported non-regular file: %s (mode %v)", path, d.Type()))
			return nil
		}

		// 确保父目录存在
		if err = os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			errs = append(errs, fmt.Sprintf("mkdir parent %s: %v", target, err))
			return nil
		}

		// 已存在文件处理
		if _, statErr := os.Stat(target); statErr == nil {
			if !force {
				errs = append(errs, fmt.Sprintf("exists %s (use --force to overwrite)", target))
				return nil
			}
		}

		rf, err := src.Open(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("open %s: %v", path, err))
			return nil
		}
		defer func() {
			if cerr := rf.Close(); cerr != nil {
				errs = append(errs, fmt.Sprintf("close src %s: %v", path, cerr))
			}
		}()

		wf, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			errs = append(errs, fmt.Sprintf("create %s: %v", target, err))
			return nil
		}
		if _, err := io.Copy(wf, rf); err != nil {
			errs = append(errs, fmt.Sprintf("copy %s: %v", path, err))
		}
		if cerr := wf.Close(); cerr != nil {
			errs = append(errs, fmt.Sprintf("close dest %s: %v", target, cerr))
		}
		return nil
	})

	if walkErr != nil {
		errs = append(errs, fmt.Sprintf("walk error: %v", walkErr))
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
