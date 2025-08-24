package newproject

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
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
}

var builtinGoTemplates = map[string]GoFileTemplate{
	"empty":   {Path: "", Type: "empty"},
	"basic":   {Path: "basic", Type: "embed"},
	"std-web": {Path: "std-web", Type: "embed"},
}

// NewGoInitOptions 创建一个新的 GoInitOptions 实例
func NewGoInitOptions() GoInitOptions {
	return GoInitOptions{Templates: builtinGoTemplates}
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
		Path: path,
		Type: templateType,
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
