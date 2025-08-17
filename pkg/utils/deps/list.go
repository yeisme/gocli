package deps

import "github.com/yeisme/gocli/pkg/tools"

// RunGoModList 执行 `go list -m` 相关命令以列出模块依赖，支持：
//   - JSON: 追加 `-json`，以 JSON 结构输出（每个模块一段 JSON）；
//   - Update: 追加 `-u`，显示"可更新至的版本"，例如 `v1.0.0 [v1.2.3]`；
//   - args: 作为目标模块/模式，默认 "all"（等价于 `go list -m all`）.
//
// 常见等价:
//
//	RunGoModList(nil, {JSON:false, Update:true})  => go list -m -u all
//	RunGoModList([]string{"all"}, {...})        => go list -m all
//	RunGoModList([]string{"std"}, {...})        => go list -m std
//
// 返回标准输出字符串；若命令失败，错误中包含 stderr 详情.
func RunGoModList(args []string, option struct {
	JSON   bool
	Update bool
}) (string, error) {
	// Always start with base command
	base := []string{"list", "-m"}
	if option.JSON {
		base = append(base, "-json")
	}
	if option.Update {
		base = append(base, "-u")
	}
	// Treat incoming args as targets/patterns
	if len(args) == 0 {
		base = append(base, "all")
	} else {
		base = append(base, args...)
	}

	output, err := tools.NewExecutor("go", base...).Output()
	if err != nil {
		return "", err
	}
	return output, nil

}
