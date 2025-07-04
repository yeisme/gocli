package cmd

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"

	"embed"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/utils"
)

var (
	templateData = map[string]string{
		"User":        utils.GetCurrentUser(),
		"Manager":     utils.GetManager(manager),
		"ManagerFile": utils.GetManagerFile(manager),
		"ProjectName": utils.GetProjectName(),
		"Author":      utils.GetAuthor(),
		"PluginsHome": utils.GetPluginsHome(),
	}

	configCmd = &cobra.Command{
		Use:   "config",
		Short: "A brief description of your command",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	setEdit       string
	configEditCmd = &cobra.Command{
		Use:   "edit",
		Short: "Edit the configuration file",
		Run: func(cmd *cobra.Command, args []string) {
			u := utils.IsUser()

			// 设置编辑器
			if setEdit != "" {
				if err := utils.SetEditor(setEdit); err != nil {
					utils.Error("Error setting editor '%s': %v\n", setEdit, err)
					return
				}
			} else {
				if err := utils.SetEditor(""); err != nil {
					utils.Error("No editor found. Please set EDITOR environment variable or install one of: code, cursor, vim, nvim, nano, subl\n")
					return
				}
			}

			editor, err := utils.Editor()
			if err != nil || editor == "" {
				utils.Error("Error finding editor: %v\n", err)
				return
			}

			var configPath string
			if u {
				configPath = utils.UserConfigPath()
			} else {
				configPath = utils.ProjectConfigPath(utils.GetConfigFile())
				if configPath == "" {
					utils.Error("No project configuration file found. Expected .gocli.yaml or .gocli.yml in current directory\n")
					return
				}
			}

			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				if u {
					utils.Error("User configuration file does not exist: %s\n", configPath)
				} else {
					utils.Error("Project configuration file does not exist: %s\n", configPath)
				}
				return
			}

			if utils.IsVerbose() {
				utils.Debug("Opening configuration file: %s with editor: %s\n", configPath, editor)
			}
			if err := exec.Command(editor, configPath).Run(); err != nil {
				utils.Error("Error opening config file with editor: %v\n", err)
				return
			}
			if !utils.IsQuiet() {
				cmd.Printf("Configuration file edited successfully: %s\n", configPath)
			}
		},
	}

	//go:embed templates
	example       embed.FS
	force         bool
	templateFile  string
	manager       string
	configInitCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize a new configuration file",
		Long: `Initialize a new configuration file for gocli.

Available templates:
  default - Basic configuration template
  user    - User-specific template (for --user flag)
  project - Project-specific template

Template lookup priority:
  1. External templates in ~/.gocli/templates/ (e.g., ~/.gocli/templates/user.yaml)
  2. Built-in embedded templates

Custom templates:
  You can place custom templates in ~/.gocli/templates/ directory.
  For example: ~/.gocli/templates/mytemplate.yaml
  Then use: gocli config init mytemplate

If no template is specified, 'default' will be used for project configs
and 'user' template will be used with --user flag.`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := initConfigFile(args); err != nil {
				utils.Error("Failed to initialize configuration: %v\n", err)
				return
			}
		},
	}

	configListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all available templates",
		Long: `List all available configuration templates.

This command shows both built-in templates and custom templates
from your ~/.gocli/templates/ directory.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := listTemplates(); err != nil {
				utils.Error("Failed to list templates: %v\n", err)
				return
			}
		},
	}
)

// initConfigFile initializes a new configuration file
func initConfigFile(args []string) error {
	v := utils.IsVerbose()
	u := utils.IsUser()

	// 动态更新模板数据，确保使用正确的 manager 值
	if manager != "" {
		templateData["Manager"] = utils.GetManager(manager)
		templateData["ManagerFile"] = utils.GetManagerFile(manager)
		if v {
			utils.Debug("Using manager: %s, manager file: %s\n", templateData["Manager"], templateData["ManagerFile"])
		}
	}

	// 确定模板类型
	templateType := templateFile
	if len(args) > 0 {
		templateType = args[0]
	}
	if templateType == "" {
		if u {
			templateType = "user"
		} else {
			templateType = "default"
		}
	}

	// 获取配置文件路径
	var configPath string
	if u {
		configPath = utils.UserConfigPath()
		if v {
			utils.Debug("Using user config path: %s\n", configPath)
		}
	} else {
		configPath = utils.ProjectConfigPath(utils.GetConfigFile())
		if v {
			utils.Debug("Using project config path: %s\n", configPath)
		}
	}

	// 检查配置文件是否已存在
	if _, err := os.Stat(configPath); err == nil && !force {
		return fmt.Errorf("configuration file already exists: %s\nUse --force to overwrite", configPath)
	}

	// 读取模板内容
	templateContent, err := getTemplateContent(templateType, u, v)
	if err != nil {
		return fmt.Errorf("failed to get template content: %w", err)
	}

	// 确保目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 写入配置文件
	if err := os.WriteFile(configPath, templateContent, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// 成功消息
	if !utils.IsQuiet() {
		utils.Success("Configuration file created successfully: %s", configPath)
		if v {
			utils.Info("Template used: %s", templateType)
		}

		// 提供下一步建议
		if u {
			utils.Info("You can now edit your user configuration with: gocli config edit --user")
		} else {
			utils.Info("You can now edit your project configuration with: gocli config edit")
		}
	}

	return nil
}

// getGocliTemplatesDir returns the path to the gocli templates directory
func getGocliTemplatesDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".gocli", "templates")
}

// readTemplateFromPath tries to read template from external path first, then falls back to embedded
func readTemplateFromPath(templateName string, verbose bool) ([]byte, error) {
	var templateContent []byte

	// 首先尝试从用户的 ~/.gocli/templates 目录读取
	externalTemplatePath := filepath.Join(getGocliTemplatesDir(), templateName)
	if _, err := os.Stat(externalTemplatePath); err == nil {
		if verbose {
			utils.Debug("Using external template: %s\n", externalTemplatePath)
		}
		templateContent, err = os.ReadFile(externalTemplatePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read external template: %w", err)
		}
	} else {
		// 回退到内置模板
		embeddedPath := fmt.Sprintf("templates/%s", templateName)
		if verbose {
			utils.Debug("Using embedded template: %s\n", embeddedPath)
		}
		templateContent, err = example.ReadFile(embeddedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded template: %w", err)
		}
	}

	// 使用 text/template 解析模板
	tmpl, err := template.New(templateName).Parse(string(templateContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// 执行模板并生成最终内容
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// getTemplateContent retrieves template content based on template type
func getTemplateContent(templateType string, isUser, verbose bool) ([]byte, error) {
	var templateContent []byte
	var err error

	switch templateType {
	case "user":
		templateContent, err = readTemplateFromPath("user.yaml", verbose)
		if err != nil {
			return nil, fmt.Errorf("failed to read user template: %w", err)
		}

	case "project", "default":
		if !isUser {
			// 首先检查是否有用户配置可以作为模板
			userConfigPath := utils.UserConfigPath()
			if _, statErr := os.Stat(userConfigPath); statErr == nil {
				templateContent, err = os.ReadFile(userConfigPath)
				if err != nil {
					return nil, fmt.Errorf("failed to read user config as template: %w", err)
				}
				if verbose {
					utils.Debug("Using user config as template: %s\n", userConfigPath)
				}
			} else {
				// 用户配置不存在，使用默认模板
				templateContent, err = readTemplateFromPath("gocli.yaml", verbose)
				if err != nil {
					return nil, fmt.Errorf("failed to read default template: %w", err)
				}
			}
		} else {
			// 对于用户配置，使用用户模板
			templateContent, err = readTemplateFromPath("user.yaml", verbose)
			if err != nil {
				return nil, fmt.Errorf("failed to read user template: %w", err)
			}
		}
	default:
		// 尝试读取自定义模板文件
		customTemplateName := fmt.Sprintf("%s.yaml", templateType)
		templateContent, err = readTemplateFromPath(customTemplateName, verbose)
		if err != nil {
			return nil, fmt.Errorf("unknown template type '%s'. Available templates: default, user, project, or place custom templates in %s", templateType, getGocliTemplatesDir())
		}
		if verbose {
			utils.Debug("Using custom template: %s\n", templateType)
		}
	}

	return templateContent, nil
}

// listTemplates lists all available templates
func listTemplates() error {
	v := utils.IsVerbose()

	if !utils.IsQuiet() {
		utils.Header("Available Configuration Templates")
	}

	// 显示内置模板
	if !utils.IsQuiet() {
		utils.SubHeader("Built-in Templates")
		builtinTemplates := []struct {
			name        string
			description string
		}{
			{"default", "Basic project configuration template"},
			{"user", "User-specific global configuration template"},
		}

		for _, template := range builtinTemplates {
			utils.ListItem("%s - %s", template.name, template.description)
		}
	}

	// 扫描外部模板
	templatesDir := getGocliTemplatesDir()
	if _, err := os.Stat(templatesDir); err == nil {
		entries, err := os.ReadDir(templatesDir)
		if err != nil {
			return fmt.Errorf("failed to read templates directory: %w", err)
		}

		var customTemplates []struct {
			name string
			path string
		}
		for _, entry := range entries {
			if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml") {
				templateName := entry.Name()[:len(entry.Name())-len(filepath.Ext(entry.Name()))]
				templatePath := filepath.Join(templatesDir, entry.Name())
				customTemplates = append(customTemplates, struct {
					name string
					path string
				}{templateName, templatePath})
			}
		}

		if len(customTemplates) > 0 && !utils.IsQuiet() {
			utils.SubHeader("Custom Templates")
			utils.Info("Found in: %s", templatesDir)

			for _, template := range customTemplates {
				utils.ListItem("%s", template.name)
				if v {
					utils.Debug("  Path: %s", template.path)
				}
			}
		} else if len(customTemplates) == 0 && v {
			utils.Debug("No custom templates found in %s", templatesDir)
		}
	} else {
		if v {
			utils.Debug("Templates directory does not exist: %s", templatesDir)
		}
	}
	return nil
}

func init() {
	configEditCmd.Flags().StringVarP(&setEdit, "editor", "e", "", "Set the editor to use for editing the configuration file")

	configInitCmd.Flags().BoolVarP(&force, "force", "", false, "Force overwrite existing configuration file")
	configInitCmd.Flags().StringVarP(&templateFile, "template", "t", "", "Template to use (default, user, project)")
	configInitCmd.Flags().StringVarP(&manager, "manager", "m", "", "Specify the manager to use (e.g., task,just,make...)")

	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(
		configEditCmd,
		configInitCmd,
		configListCmd,
	)
}
