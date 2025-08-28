// Package main provides the entry point for the gocli schema generation.
package main

import (
	"os"

	"github.com/yeisme/gocli/pkg/utils/schema"
)

//go:generate go run github.com/yeisme/gocli/cmd/schema
func main() {
	if _, err := os.Stat("../../docs"); os.IsNotExist(err) {
		if err := os.Mkdir("../../docs", 0755); err != nil {
			panic(err)
		}
	}

	toolsSchemaFile, err := os.Create("../../docs/tools_schema.json")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = toolsSchemaFile.Close()
	}()

	if err = schema.GenToolsSchema(toolsSchemaFile); err != nil {
		panic(err)
	}

	configSchemaFile, err := os.Create("../../docs/config_schema.json")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = configSchemaFile.Close()
	}()

	if err := schema.GenConfigSchema(configSchemaFile); err != nil {
		panic(err)
	}
}
