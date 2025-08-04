package tools

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// 测试基础命令执行
func TestExecutor_Run(t *testing.T) {
	   e := NewExecutor("echo", "hello world")
	   stdout, stderr, err := e.Run()
	   if err != nil {
			   t.Fatalf("Run failed: %v", err)
	   }
	   if !strings.Contains(stdout, "hello world") {
			   t.Errorf("stdout should contain 'hello world', got: %q", stdout)
	   }
	   if stderr != "" {
			   t.Errorf("stderr should be empty, got: %q", stderr)
	   }
}

// 测试 Output 方法
func TestExecutor_Output(t *testing.T) {
	   e := NewExecutor("echo", "foo bar")
	   out, err := e.Output()
	   if err != nil {
			   t.Fatalf("Output failed: %v", err)
	   }
	   if !strings.Contains(out, "foo bar") {
			   t.Errorf("output should contain 'foo bar', got: %q", out)
	   }
}

// 测试 WithDir
func TestExecutor_WithDir(t *testing.T) {
	   dir := t.TempDir()
	   var e *Executor
	   if runtime.GOOS == "windows" {
			   e = NewExecutor("cmd", "/c", "cd")
	   } else {
			   e = NewExecutor("pwd")
	   }
	   e.WithDir(dir)
	   stdout, _, err := e.Run()
	   if err != nil {
			   t.Fatalf("Run with dir failed: %v", err)
	   }
	   got := strings.TrimSpace(stdout)
	   want, _ := filepath.Abs(dir)
	   if !strings.EqualFold(got, want) {
			   t.Errorf("expected working directory to be %q, got %q", want, got)
	   }
}

// 测试 WithEnv
func TestExecutor_WithEnv(t *testing.T) {
	   var e *Executor
	   if runtime.GOOS == "windows" {
			   e = NewExecutor("powershell", "-Command", "$env:FOO")
	   } else {
			   e = NewExecutor("sh", "-c", "echo $FOO")
	   }
	   e.WithEnv("FOO=bar_test_env")
	   stdout, _, err := e.Run()
	   if err != nil {
			   t.Fatalf("Run with env failed: %v", err)
	   }
	   if !strings.Contains(stdout, "bar_test_env") {
			   t.Errorf("stdout should contain 'bar_test_env', got: %q", stdout)
	   }
}

// 测试 WithStdin
func TestExecutor_WithStdin(t *testing.T) {
	   var e *Executor
	   if runtime.GOOS == "windows" {
			   e = NewExecutor("findstr", "/n", "^")
	   } else {
			   e = NewExecutor("cat")
	   }
	   e.WithStdin(strings.NewReader("input from stdin"))
	   stdout, _, err := e.Run()
	   if err != nil {
			   t.Fatalf("Run with stdin failed: %v", err)
	   }
	   if !strings.Contains(stdout, "input from stdin") {
			   t.Errorf("stdout should contain 'input from stdin', got: %q", stdout)
	   }
}

// 测试命令不存在时的错误处理
func TestExecutor_Run_Error(t *testing.T) {
	   e := NewExecutor("not_a_real_command_12345")
	   _, _, err := e.Run()
	   if err == nil {
			   t.Fatal("expected error for nonexistent command, got nil")
	   }
	   if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "executable file not found") {
			   t.Errorf("error should indicate command not found, got: %v", err)
	   }
}
