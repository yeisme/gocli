// Package executor 提供了一个用于执行外部命令的工具，支持捕获输出、错误处理和流式输出等功能
package executor

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
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
	args := strings.Join(e.Args, " ")

	// 将字面 "\\n" 转为真实换行，便于多行显示
	// 使用 CleanStderr 得到已经去 ANSI、修整过的 stderr
	stderr := strings.TrimSpace(strings.ReplaceAll(e.CleanStderr(), `\\n`, "\n"))

	// 尝试获取数值类型的退出码
	code := e.ExitCode()
	codeStr := "unknown"
	if code >= 0 {
		codeStr = fmt.Sprintf("%d", code)
	}

	if stderr == "" {
		return fmt.Sprintf("command execution failed: %s %s, exit-code: %s, err: %v", e.Cmd, args, codeStr, e.Err)
	}

	// 按行缩进 stderr，增强可读性
	lines := strings.Split(stderr, "\n")
	for i, l := range lines {
		lines[i] = "\t" + l
	}

	return fmt.Sprintf("command execution failed: %s %s, exit-code: %s, err: %v\nstderr:\n%s",
		e.Cmd, args, codeStr, e.Err, strings.Join(lines, "\n"))
}

// Unwrap 允许使用 errors.Is 和 errors.As 来检查底层错误
func (e *ExecError) Unwrap() error {
	return e.Err
}

// ansiRegexp 用于匹配 ANSI 颜色/格式化控制序列，例如 "\x1b[31m"
var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// CleanStderr 返回去除 ANSI 控制码并修整空白的 stderr 文本
func (e *ExecError) CleanStderr() string {
	if strings.TrimSpace(e.Stderr) == "" {
		return ""
	}
	s := ansiRegexp.ReplaceAllString(e.Stderr, "")
	return strings.TrimSpace(s)
}

// ExitCode 返回底层进程的退出码，若不可用返回 -1
func (e *ExecError) ExitCode() int {
	if exitErr, ok := e.Err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return -1
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

// WithStdout 设置命令的标准输出
func (e *Executor) WithStdout(w io.Writer) *Executor {
	e.cmd.Stdout = w
	return e
}

// WithStderr 设置命令的标准错误输出
func (e *Executor) WithStderr(w io.Writer) *Executor {
	e.cmd.Stderr = w
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

// RunStreaming 执行命令并将标准输出/错误流式写入提供的 io.Writer.
// 为了在出错时仍能返回 stderr 内容，会在内部附加一个缓冲区捕获 stderr.
// 仅在返回错误时，错误中的 Stderr 才会包含该缓冲区内容.
func (e *Executor) RunStreaming(stdout, stderr io.Writer) error {
	var errBuf bytes.Buffer

	if stdout != nil {
		e.cmd.Stdout = stdout
	}
	// 确保在写入外部 stderr 的同时也能捕获错误信息
	switch {
	case stderr != nil && e.cmd.Stderr != nil && e.cmd.Stderr != stderr:
		e.cmd.Stderr = io.MultiWriter(e.cmd.Stderr, stderr, &errBuf)
	case stderr != nil:
		e.cmd.Stderr = io.MultiWriter(stderr, &errBuf)
	default:
		// 即使没有外部 stderr，也捕获到缓冲区，便于错误返回
		e.cmd.Stderr = &errBuf
	}

	if err := e.cmd.Run(); err != nil {
		return &ExecError{
			Cmd:    e.cmd.Path,
			Args:   e.cmd.Args[1:],
			Stderr: errBuf.String(),
			Err:    err,
		}
	}
	return nil
}
