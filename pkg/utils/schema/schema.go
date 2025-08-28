// Package schema provides utilities for working with JSON schemas.
package schema

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/invopop/jsonschema"
	"github.com/yeisme/gocli/pkg/configs"
	"github.com/yeisme/gocli/pkg/tools"
)

// GenToolsSchema generates the JSON schema for the tools configuration and writes it to the provided writer.
func GenToolsSchema(out io.Writer) error {
	reflector := &jsonschema.Reflector{
		FieldNameTag:               "mapstructure",
		RequiredFromJSONSchemaTags: true,
	}
	toolSchema := reflector.Reflect(map[string]tools.InstallToolsInfo{})
	schemaJSON, err := json.MarshalIndent(toolSchema, "", "  ")
	if err != nil {
		return err
	}

	fmt.Fprintln(out, string(schemaJSON))
	return nil
}

// GenConfigSchema generates the JSON schema for the entire application configuration and writes it to the provided writer.
func GenConfigSchema(out io.Writer) error {
	reflector := &jsonschema.Reflector{
		AllowAdditionalProperties:  true,
		RequiredFromJSONSchemaTags: true,
		FieldNameTag:               "mapstructure",
	}
	configSchema := reflector.Reflect(configs.Config{})
	schemaJSON, err := json.MarshalIndent(configSchema, "", "  ")
	if err != nil {
		return err
	}

	fmt.Fprintln(out, string(schemaJSON))
	return nil
}
