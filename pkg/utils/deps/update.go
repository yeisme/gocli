// Package deps provides utilities for managing Go module dependencies.
package deps

import "github.com/yeisme/gocli/pkg/tools"

// RunGoUpdate 执行 `go get -u <targets>` 以更新模块依赖到最新次要/补丁版本
//
// 行为说明:
//   - 当 args 为空(nil)时，默认使用 `./...`，即更新当前模块内所有包涉及到的依赖；
//   - 返回标准输出字符串；失败时错误中包含 stderr 详情；
//   - 等价示例：
//     RunGoUpdate(nil)                          => go get -u ./...
//     RunGoUpdate([]string{"all"})            => go get -u all
//     RunGoUpdate([]string{"example.com/mod"}) => go get -u example.com/mod
func RunGoUpdate(args []string) (string, error) {
	if args == nil {
		args = []string{"./..."} // Default to updating all dependencies
	}
	output, err := tools.NewExecutor("go", append([]string{"get", "-u"}, args...)...).Output()
	if err != nil {
		return "", err
	}
	return output, nil
}
