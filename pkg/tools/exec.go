// tools/executor.go
package tools

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// ExecError 是一个结构化的命令执行错误，包含了丰富的上下文信息
type ExecError struct {
	Cmd    string   // 执行的命令
	Args   []string // 命令参数
	Stderr string   // 标准错误输出
	Err    error    // 底层错误 (通常是 *exec.ExitError)
}

// Error 实现了 error 接口，返回一个详细的错误信息
func (e *ExecError) Error() string {
	return fmt.Sprintf("command execution failed: %s %s\nerror: %v\nstderr: %s",
		e.Cmd, strings.Join(e.Args, " "), e.Err, strings.TrimSpace(e.Stderr))
}

// Unwrap 允许使用 errors.Is 和 errors.As 来检查底层错误
func (e *ExecError) Unwrap() error {
	return e.Err
}

// Executor 是一个命令执行器的构建器
// 它采用链式调用来配置命令，最终通过 Run, Output 等方法执行
// 一个 Executor 实例应该用于一次命令执行
type Executor struct {
	cmd *exec.Cmd
}

// NewExecutor 创建一个新的命令执行器
func NewExecutor(name string, args ...string) *Executor {
	return &Executor{
		cmd: exec.Command(name, args...),
	}
}

// WithDir 设置命令执行的工作目录
func (e *Executor) WithDir(dir string) *Executor {
	e.cmd.Dir = dir
	return e
}

// WithStdin 设置命令的标准输入
func (e *Executor) WithStdin(r io.Reader) *Executor {
	e.cmd.Stdin = r
	return e
}

// WithEnv 附加环境变量到命令
// 它会附加到当前进程的环境变量之上
func (e *Executor) WithEnv(envs ...string) *Executor {
	e.cmd.Env = append(e.cmd.Environ(), envs...)
	return e
}

// Run 执行命令，并分别返回标准输出和标准错误
// 即使命令执行失败，stdout 和 stderr 也会返回捕获到的内容
func (e *Executor) Run() (stdout, stderr string, err error) {
	var outBuf, errBuf bytes.Buffer
	e.cmd.Stdout = &outBuf
	e.cmd.Stderr = &errBuf

	runErr := e.cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if runErr != nil {
		err = &ExecError{
			Cmd:    e.cmd.Path,
			Args:   e.cmd.Args[1:],
			Stderr: stderr,
			Err:    runErr,
		}
	}

	return stdout, stderr, err
}

// Output 执行命令并返回其标准输出
// 如果发生错误，错误信息中会包含标准错误的内容
func (e *Executor) Output() (string, error) {
	output, err := e.cmd.Output()
	if err != nil {
		// *exec.ExitError 已经包含了 Stderr
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(output), &ExecError{
				Cmd:    e.cmd.Path,
				Args:   e.cmd.Args[1:],
				Stderr: string(exitErr.Stderr),
				Err:    err,
			}
		}
		return string(output), &ExecError{
			Cmd:  e.cmd.Path,
			Args: e.cmd.Args[1:],
			Err:  err,
		}
	}
	return string(output), nil
}

// CombinedOutput 执行命令并返回其合并的标准输出和标准错误
func (e *Executor) CombinedOutput() (string, error) {
	output, err := e.cmd.CombinedOutput()
	if err != nil {
		// CombinedOutput 的 Stderr 已经混入 output 中
		return string(output), &ExecError{
			Cmd:    e.cmd.Path,
			Args:   e.cmd.Args[1:],
			Stderr: string(output),
			Err:    err,
		}
	}
	return string(output), nil
}
