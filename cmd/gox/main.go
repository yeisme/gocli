// Package main implements the gox command
package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/context"
	toolsPkg "github.com/yeisme/gocli/pkg/tools"
	log2 "github.com/yeisme/gocli/pkg/utils/log"
)

var (
	gocliCtx    *context.GocliContext
	globalFlags = context.GlobalFlags{}
	configPath  = globalFlags.ConfigPath
	log         log2.Logger

	gox = cobra.Command{
		Use:   "gox <tool> [args...]",
		Short: toolsPkg.ToolsRunMsg.Short,
		Long:  toolsPkg.ToolsRunMsg.Long,
		// Disable cobra flag parsing so flags after the tool name (including
		// --help) are forwarded verbatim to the executed binary. We still want
		// `gocli tools run --help` to show the run command help, so detect that
		// specific form in PreRun and print help for the run command.
		DisableFlagParsing: true,
		PreRun: func(cmd *cobra.Command, _ []string) {
			ctx := context.InitGocliContext(configPath, false, false, true)

			gocliCtx = ctx
			log = ctx.Logger

			log.Info().Msgf("Execute Command: %s %s", "gocli", strings.Join(os.Args[1:], " "))

			toolsPkg.ShowRunHelpIfRequested(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			gocliToolsPath := gocliCtx.Config.Tools.GoCLIToolsPath
			if err := toolsPkg.ExecuteToolRun(args, cmd.OutOrStdout(), false, gocliToolsPath); err != nil {
				log.Error().Err(err).Msg("failed to execute tool")
			}
		},
		// allow unknown flags so that flags intended for the executed tool
		// are not treated as errors by cobra and can be forwarded.
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		Aliases:            []string{"x", "exec"},
	}
)

func main() {
	if err := gox.Execute(); err != nil {
		log.Error().Err(err).Msg("failed to execute gocli")
	}
}

func init() {
	toolsPkg.SetRunHelpFunc(&gox)
}
