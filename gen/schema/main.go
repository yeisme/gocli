package main

//go:generate go run github.com/yeisme/gocli/gen/schema

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/invopop/jsonschema"
	"github.com/yeisme/gocli/pkg/types"
)

func main() {
	schema := jsonschema.ReflectFromType(reflect.TypeOf(types.Config{}))
	file, err := os.OpenFile("schema.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(schema); err != nil {
		fmt.Println("Error encoding JSON:", err)
	}
}
