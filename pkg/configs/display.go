package configs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yeisme/gocli/pkg/style"
	"gopkg.in/yaml.v3"
)

// OutputFormat 输出格式类型
type OutputFormat string

const (
	// FormatYAML represents the YAML output format.
	FormatYAML OutputFormat = "yaml"
	// FormatJSON represents the JSON output format.
	FormatJSON OutputFormat = "json"
	// FormatTOML represents the TOML output format.
	FormatTOML OutputFormat = "toml"
	// FormatText represents the plain text output format.
	FormatText OutputFormat = "text"
	// FormatTable represents the table output format.
	FormatTable OutputFormat = "table"
)

// ValidFormats 返回所有有效的输出格式
func ValidFormats() []string {
	return []string{string(FormatYAML), string(FormatJSON), string(FormatTOML), string(FormatText), string(FormatTable)}
}

// ParseOutputFormat 解析输出格式字符串
func ParseOutputFormat(format string) (OutputFormat, error) {
	switch strings.ToLower(format) {
	case "yaml", "yml":
		return FormatYAML, nil
	case "json":
		return FormatJSON, nil
	case "toml":
		return FormatTOML, nil
	case "text", "txt":
		return FormatText, nil
	case "table":
		return FormatTable, nil
	default:
		return "", fmt.Errorf("unsupported format '%s', supported formats: %s", format, strings.Join(ValidFormats(), ", "))
	}
}

// GetOutputFormatFromFlags 从命令行标志获取输出格式
func GetOutputFormatFromFlags(cmd *cobra.Command) OutputFormat {
	// 首先检查 --format 标志
	if formatFlag, _ := cmd.Flags().GetString("format"); formatFlag != "" {
		if format, err := ParseOutputFormat(formatFlag); err == nil {
			return format
		}
	}

	// 检查具体的格式标志
	if yaml, _ := cmd.Flags().GetBool("yaml"); yaml {
		return FormatYAML
	}
	if jsonFlag, _ := cmd.Flags().GetBool("json"); jsonFlag {
		return FormatJSON
	}
	if toml, _ := cmd.Flags().GetBool("toml"); toml {
		return FormatTOML
	}
	if text, _ := cmd.Flags().GetBool("text"); text {
		return FormatText
	}
	if table, _ := cmd.Flags().GetBool("table"); table {
		return FormatTable
	}

	// 默认格式
	return FormatYAML
}

// OutputData 根据指定格式输出数据
func OutputData(data any, format OutputFormat, out io.Writer, color bool) error {
	switch format {
	case FormatYAML:
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		err := enc.Encode(data)
		if err != nil {
			return fmt.Errorf("failed to marshal to YAML: %w", err)
		}
		err = enc.Close()
		if err != nil {
			return fmt.Errorf("failed to close YAML encoder: %w", err)
		}
		if color {
			_ = style.PrintYAML(out, buf.String())
		} else {
			fmt.Fprint(out, buf.String())
		}
	case FormatJSON:
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal to JSON: %w", err)
		}
		if color {
			_ = style.PrintJSON(out, jsonData)
		} else {
			fmt.Fprint(out, string(jsonData))
		}
	case FormatTOML:
		tomlData, err := toml.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal to TOML: %w", err)
		}
		if color {
			_ = style.PrintTOML(out, string(tomlData))
		} else {
			fmt.Fprint(out, string(tomlData))
		}
	case FormatText:
		// 简单的文本格式输出
		fmt.Fprintf(out, "%+v\n", data)

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return nil
}

// GetConfigSection 从 viper 实例获取指定配置段
func GetConfigSection(v *viper.Viper, section string, showAll bool) (any, error) {
	if showAll {
		// 返回完整的配置结构体（包含默认值）
		var config Config
		if err := v.Unmarshal(&config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		if section == "" {
			return config, nil
		}

		// 支持点分层次路径，比如 "app.hotload"
		parts := strings.Split(section, ".")
		// 从 config 的反射值开始递归查找
		var curVal = reflect.ValueOf(config)

		for _, part := range parts {
			for curVal.Kind() == reflect.Pointer || curVal.Kind() == reflect.Interface {
				curVal = curVal.Elem()
			}

			if !curVal.IsValid() {
				return nil, fmt.Errorf("unknown configuration section: %s", section)
			}

			// only structs can have tagged fields
			if curVal.Kind() != reflect.Struct {
				return nil, fmt.Errorf("configuration section is not a struct: %s", part)
			}

			typ := curVal.Type()
			found := false
			lowerPart := strings.ToLower(part)

			for i := 0; i < curVal.NumField(); i++ {
				field := typ.Field(i)
				tag := field.Tag.Get("mapstructure")

				// mapstructure tag may contain options like `mapstructure:"name,omitempty"`
				if idx := strings.Index(tag, ","); idx != -1 {
					tag = tag[:idx]
				}

				if tag == "" {
					// fallback to field name
					tag = field.Name
				}

				if strings.ToLower(tag) == lowerPart {
					curVal = curVal.Field(i)
					found = true
					break
				}
			}

			if !found {
				return nil, fmt.Errorf("unknown configuration section: %s", section)
			}
		}

		// 解引用并返回最终值的接口表示
		for curVal.Kind() == reflect.Pointer || curVal.Kind() == reflect.Interface {
			curVal = curVal.Elem()
		}

		if !curVal.IsValid() {
			return nil, fmt.Errorf("unknown configuration section: %s", section)
		}

		return curVal.Interface(), nil
	}

	// 返回 viper 的原始数据
	lowerSection := strings.ToLower(section)

	if lowerSection == "" {
		// 显示所有配置
		return v.AllSettings(), nil
	}

	// 检查 section 是否是 viper 中的一个顶级键或已设置的键
	if v.IsSet(lowerSection) {
		return v.Get(lowerSection), nil
	}

	return nil, fmt.Errorf("unknown or unset configuration section %s", section)

}
