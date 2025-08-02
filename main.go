package main

import (
	"github.com/yeisme/gocli/cmd"
	"github.com/yeisme/gocli/pkg/context"
	"github.com/yeisme/gocli/pkg/utils"
)

func main() {
	ctx := context.InitGocliContext()
	// 执行命令
	utils.Info().Str("key", "value").Msg("Starting application")
	cmd.Execute(ctx)
}
