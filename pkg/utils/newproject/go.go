package newproject

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yeisme/gocli/pkg/utils/executor"
)

// GoInitOptions golang 项目的初始化配置选项
type GoInitOptions struct {
	// TemplateType 模板类型
	// 短名 builtinGoTemplates 中的键（例如 "basic", "empty" etc.）
	// 或直接指向一个包含 go.mod 的目录/仓库
	// 默认: "empty"
	Templates map[string]GoFileTemplate `json:"templates"`
}

// GoFileTemplate 用于指定 Go 模板文件的配置
type GoFileTemplate struct {
	// Path 用于指定模板文件的路径
	// 例如 `stdweb` 对应 templates/go/stdweb
	Path string `json:"path"`

	// Type embed/http(s)/git/file_system
	Type string `json:"type"`

	// Language 模板所属编程语言（可选，默认 go）
	Language string `json:"language"`
}

var builtinGoTemplates = map[string]GoFileTemplate{
	"empty":  {Path: "", Type: "empty", Language: "go"},
	"basic":  {Path: "basic", Type: "embed", Language: "go"},
	"stdweb": {Path: "stdweb", Type: "embed", Language: "go"},
}

// NewGoInitOptions 创建一个新的 GoInitOptions 实例
func NewGoInitOptions() GoInitOptions {
	return GoInitOptions{Templates: builtinGoTemplates}
}

// GetGoTemplateFS 返回 Go 模板的文件系统，定位到 templates/go/<name>
func GetGoTemplateFS(name string, opts InitOptions) (fs.FS, error) {
	if name == "" {
		return nil, fmt.Errorf("template name is empty")
	}

	templateInfo, ok := opts.Go.Templates[name]
	if !ok {
		return nil, fmt.Errorf("template %q not found in options", name)
	}

	switch strings.ToLower(templateInfo.Type) {
	case "embed":
		return GetEmbeddedGoTemplate(name, opts)
	case "empty":
		return nil, nil
	case "http", "https":
		return GetHTTPGoTemplate(name, opts)
	case "git":
		return GetGitGoTemplate(name, opts)
	case "file_system", "filesystem", "fs":
		return GetFileSystemGoTemplate(name, opts)
	default:
		return nil, fmt.Errorf("unsupported template type %q for %q", templateInfo.Type, name)
	}
}

// GetEmbeddedGoTemplate 返回嵌入模板中的子文件系统，定位到 templates/go/<name>
// 例如 name="basic" 将返回 embedTemplateFS 的 templates/go/basic 子 FS
func GetEmbeddedGoTemplate(name string, opts InitOptions) (fs.FS, error) {
	if name == "" {
		return nil, fmt.Errorf("template name is empty")
	}
	templateInfo := opts.Go.Templates[name]
	// use slash-separated path because embed FS uses forward slashes
	p := path.Join("templates", "go", templateInfo.Path)
	sub, err := fs.Sub(embedTemplateFS, p)
	if err != nil {
		return nil, fmt.Errorf("embedded template %q not found: %w", name, err)
	}
	return sub, nil
}

// GetHTTPGoTemplate 通过 http(s) 地址（通常是 git 仓库）获取模板
// 处理逻辑：使用 git clone --depth 1 克隆到临时目录，然后返回该目录 FS
func GetHTTPGoTemplate(name string, opts InitOptions) (fs.FS, error) {
	templateInfo := opts.Go.Templates[name]
	url := templateInfo.Path
	if url == "" {
		return nil, fmt.Errorf("http template %q url is empty", name)
	}

	// 1. 创建临时目录
	workDir, err := os.MkdirTemp("", "gocli-template-http-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	// 2. 下载到临时文件
	archiveFile, err := downloadToTemp(url)
	if err != nil {
		return nil, fmt.Errorf("download template archive: %w", err)
	}

	// 3. 解压
	extractDir := filepath.Join(workDir, "extract")
	if mkErr := os.MkdirAll(extractDir, 0o755); mkErr != nil {
		return nil, fmt.Errorf("create extract dir: %w", mkErr)
	}
	if exErr := extractArchive(archiveFile, extractDir); exErr != nil {
		return nil, fmt.Errorf("extract archive: %w", exErr)
	}

	// 4. 定位包含 go.mod 的目录
	modDir, err := findGoModDir(extractDir)
	if err != nil {
		return nil, err
	}
	return os.DirFS(modDir), nil
}

// downloadToTemp 使用 http GET 下载文件到临时目录，返回文件路径
func downloadToTemp(url string) (string, error) {
	resp, err := http.Get(url) // #nosec G107: 用户配置提供的 URL（CLI 语义允许）
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http GET %s failed: status %s", url, resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "gocli-template-archive-*")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = tmpFile.Close()
	}()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return "", err
	}
	return tmpFile.Name(), nil
}

// extractArchive 根据扩展名选择 zip / tar.gz / tgz 解压
func extractArchive(archivePath, dest string) error {
	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(archivePath, dest)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(archivePath, dest)
	default:
		return fmt.Errorf("unsupported archive format: %s", archivePath)
	}
}

func extractZip(archivePath, dest string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = zr.Close()
	}()

	for _, f := range zr.File {
		targetPath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(targetPath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, f.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		func() {
			defer func() {
				_ = rc.Close()
			}()
			w, openErr := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
			if openErr != nil {
				return
			}
			defer func() {
				_ = w.Close()
			}()
			_, _ = io.Copy(w, rc)
		}()
	}
	return nil
}

func extractTarGz(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gr.Close() }()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dest, hdr.Name)
		if !strings.HasPrefix(targetPath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in tar: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, fs.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			w, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fs.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, tr); err != nil {
				_ = w.Close()
				return err
			}
			if cerr := w.Close(); cerr != nil {
				return cerr
			}
		}
	}
	return nil
}

// findGoModDir 尝试在 base 及其一级子目录中寻找 go.mod
func findGoModDir(base string) (string, error) {
	if _, err := os.Stat(filepath.Join(base, "go.mod")); err == nil {
		return base, nil
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", fmt.Errorf("read extract dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			p := filepath.Join(base, e.Name())
			if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("cannot locate go.mod under extracted template %q", base)
}

// GetGitGoTemplate 通过 git 地址获取模板（可能是 ssh/https 等）
func GetGitGoTemplate(name string, opts InitOptions) (fs.FS, error) {
	templateInfo := opts.Go.Templates[name]
	return cloneGitToTemp(templateInfo.Path)
}

// GetFileSystemGoTemplate 使用本地文件系统目录作为模板
func GetFileSystemGoTemplate(name string, opts InitOptions) (fs.FS, error) {
	templateInfo := opts.Go.Templates[name]
	p := templateInfo.Path
	if p == "" {
		return nil, fmt.Errorf("file_system template path is empty")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path for template %q: %w", name, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat template path %q: %w", abs, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("template path %q is not a directory", abs)
	}
	if err := ensureGoMod(abs); err != nil {
		return nil, err
	}
	return os.DirFS(abs), nil
}

// cloneGitToTemp 克隆一个 git 仓库（路径可以是 http(s)/ssh）
func cloneGitToTemp(repo string) (fs.FS, error) {
	if repo == "" {
		return nil, fmt.Errorf("git repository path is empty")
	}
	dir, err := os.MkdirTemp("", "gocli-template-git-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir for git template: %w", err)
	}
	// 使用 --depth 1 以提高速度
	if _, err := executor.NewExecutor("git", "clone", "--depth", "1", repo, dir).Output(); err != nil {
		return nil, fmt.Errorf("git clone %q failed: %w", repo, err)
	}
	if err := ensureGoMod(dir); err != nil {
		return nil, err
	}
	return os.DirFS(dir), nil
}

// ensureGoMod 确保目录内存在 go.mod
func ensureGoMod(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("template directory %q missing go.mod", dir)
		}
		return fmt.Errorf("check go.mod in %q: %w", dir, err)
	}
	return nil
}

// AddGoTemplateToOptions 将指定的 Go 模板添加到选项中(新增 file/http(s)/git/file_system 模板)
func AddGoTemplateToOptions(opts *InitOptions, name string, path string, templateType string) error {
	if name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	_, exists := opts.Go.Templates[name]
	if exists {
		return fmt.Errorf("template %q already exists", name)
	}

	opts.Go.Templates[name] = GoFileTemplate{
		Path:     path,
		Type:     templateType,
		Language: "go",
	}

	return nil
}

// NormalizeGoProjectName args 最多有一个，作为项目名称，当没有提供时，使用默认名称(当前目录名，例如 /home/user/myproject -> myproject)
func NormalizeGoProjectName(args []string) (string, error) {
	if len(args) == 0 {
		return getDirBase()
	}

	// 使用第一个参数作为项目名（忽略多余参数），去掉首尾空白
	raw := strings.TrimSpace(args[0])
	if raw == "" {
		return getDirBase()
	}

	// 如果传入的是类似 module 路径（包含 '/'），则保留完整路径（例如 github.com/owner/repo）
	// 并对每个 path segment 进行校验；否则只取 basename 作为短项目名
	if strings.Contains(raw, "/") {
		// 移除末尾的斜杠（如果有）
		mp := strings.TrimRight(raw, "/")
		if mp == "" {
			return "", fmt.Errorf("module path is empty")
		}
		if err := validateModulePath(mp); err != nil {
			return "", err
		}
		return mp, nil
	}

	// 否则按原逻辑只取 basename 并校验单段名称
	name := filepath.Base(raw)
	if err := validateSimpleName(name); err != nil {
		return "", err
	}
	if name == "" {
		return "", fmt.Errorf("project name is empty after normalization")
	}
	return name, nil
}

// getDirBase 返回当前工作目录的 basename，或错误
func getDirBase() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	name := filepath.Base(wd)
	if name == "." || name == string(os.PathSeparator) || name == "" {
		return "", fmt.Errorf("cannot determine project name from current directory")
	}
	return name, nil
}

// validateModulePath 校验 module path 中每个 segment 是否只包含允许的字符
func validateModulePath(mp string) error {
	segments := strings.Split(mp, "/")
	for _, seg := range segments {
		if seg == "" {
			return fmt.Errorf("invalid module path %q: empty segment", mp)
		}
		if err := validateSimpleName(seg); err != nil {
			return fmt.Errorf("invalid module path %q: segment %q invalid: %w", mp, seg, err)
		}
	}
	return nil
}

// validateSimpleName 校验单段名称（例如仓库名或本地项目名）只包含允许的字符
func validateSimpleName(name string) error {
	if name == "" {
		return fmt.Errorf("name is empty")
	}
	for _, r := range name {
		switch {
		case r == '_' || r == '-' || r == '.':
			continue
		case r >= '0' && r <= '9':
			continue
		case r >= 'A' && r <= 'Z':
			continue
		case r >= 'a' && r <= 'z':
			continue
		default:
			return fmt.Errorf("contains unsupported character %q", r)
		}
	}
	return nil
}
