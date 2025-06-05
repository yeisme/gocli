package cmd

import (
	"fmt"

	"github.com/yeisme/gocli/pkg/parse"
	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

// CommandType 表示命令类型
type CommandType string

// 定义各种命令类型常量
const (
	TypeTest    CommandType = "Test"
	TypeBuild   CommandType = "Build"
	TypeRun     CommandType = "Run"
	TypeDev     CommandType = "Dev"
	TypeClean   CommandType = "Clean"
	TypeLint    CommandType = "Lint"
	TypeRelease CommandType = "Release"
	TypeDeps    CommandType = "Deps"
	TypeHelp    CommandType = "Help"
)

// DefaultHandler 表示默认命令处理程序，当没有找到命令配置时调用
type DefaultHandler func()

// CommandProcessor 通用命令处理函数
func CommandProcessor(cmdType CommandType, args []string, listFlag bool, getCommands func(*types.Config) []types.Command) {
	ProcessorWithDefault(cmdType, args, listFlag, getCommands, nil)
}

// ProcessorWithDefault 带有默认处理程序的通用命令处理函数
func ProcessorWithDefault(cmdType CommandType, args []string, listFlag bool,
	getCommands func(*types.Config) []types.Command, defaultHandler DefaultHandler) {

	config := parse.GetProjectConfig()
	if config == nil {
		utils.Error("No project configuration found. Please run 'gocli config init' to create a configuration file.")
		return
	}

	commands := getCommands(config)
	typeName := string(cmdType)

	// Handle --list flag
	if listFlag {
		utils.Header("%s", fmt.Sprintf("Available %s Configurations", typeName))
		if len(commands) == 0 {
			utils.Info(fmt.Sprintf("No %s configurations found.", typeName))
			return
		}

		for _, cmd := range commands {
			if utils.IsVerbose() {
				utils.Box(fmt.Sprintf("%s - %s",
					cmd.Name, cmd.Description),
					fmt.Sprintf("Commands:\n%s", joinStringSlice(cmd.Cmds)),
					len(cmd.Name)+len(cmd.Description)+20)
			} else {
				utils.ListItem("%s - %s", cmd.Name, cmd.Description)
			}
		}
		return
	}

	v := utils.IsVerbose()
	if v {
		utils.Info(fmt.Sprintf("Starting %s process...", typeName))
		utils.Info(fmt.Sprintf("Loaded project configuration: %s v%s", config.Project.Name, config.Project.Version))
		utils.Info(fmt.Sprintf("Found %d %s configurations", len(commands), typeName))
	}

	// 处理没有配置的情况
	if len(commands) == 0 {
		if defaultHandler != nil {
			if v {
				utils.Info(fmt.Sprintf("No %s configurations found, using default handler", typeName))
			}
			defaultHandler()
			return
		}
		utils.Info(fmt.Sprintf("No %s configurations found.", typeName))
		return
	}

	targetName := "default"
	if len(args) > 0 {
		targetName = args[0]
		if v {
			utils.Info(fmt.Sprintf("Target %s configuration specified: %s", typeName, targetName))
		}
	} else if v {
		utils.Info(fmt.Sprintf("Using default %s configuration", typeName))
	}

	var targetCommand *types.Command
	for _, cmd := range commands {
		if cmd.Name == targetName {
			targetCommand = &cmd
			break
		}
	}

	if targetCommand == nil {
		utils.Error(fmt.Sprintf("%s configuration '%s' not found", typeName, targetName))
		if v {
			utils.Info(fmt.Sprintf("Available %s configurations:", typeName))
			for _, cmd := range commands {
				utils.Info(fmt.Sprintf("  - %s: %s", cmd.Name, cmd.Description))
			}
		}
		return
	}

	if v {
		utils.Info(fmt.Sprintf("Found %s configuration: %s", typeName, targetCommand.Name))
		utils.Info(fmt.Sprintf("Description: %s", targetCommand.Description))
		utils.Info(fmt.Sprintf("Commands to execute: %d", len(targetCommand.Cmds)))
	}

	if !utils.IsQuiet() {
		utils.Info(fmt.Sprintf("Executing %s: %s - %s", typeName, targetCommand.Name, targetCommand.Description))
	}

	for i, cmdStr := range targetCommand.Cmds {
		if v {
			utils.Info(fmt.Sprintf("Executing %s command %d/%d: %s", typeName, i+1, len(targetCommand.Cmds), cmdStr))
		}

		if err := utils.GoExec(cmdStr); err != nil {
			utils.Error(fmt.Sprintf("%s failed at command '%s': %v", typeName, cmdStr, err))
			return
		}

		if v {
			utils.Info(fmt.Sprintf("Successfully executed %s command %d/%d", typeName, i+1, len(targetCommand.Cmds)))
		}
	}

	if v {
		utils.Info(fmt.Sprintf("%s process completed successfully", typeName))
	}
}
