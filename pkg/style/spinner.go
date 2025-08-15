package style

import (
	"fmt"
	"io"
	"time"
)

// Spinner 是一个简单的终端旋转指示器
// 用于长时间运行的任务期间提供轻量反馈
type Spinner struct {
	out      io.Writer
	msg      string
	stopCh   chan struct{}
	doneCh   chan struct{}
	interval time.Duration
}

// NewSpinner 创建一个新的 Spinner
// out: 写入目标（一般为 cmd.OutOrStdout() 或 os.Stdout）
// msg: 前缀消息
func NewSpinner(out io.Writer, msg string) *Spinner {
	return &Spinner{
		out:      out,
		msg:      msg,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		interval: 120 * time.Millisecond,
	}
}

// Start 启动 spinner，直到 Stop 被调用
func (s *Spinner) Start() {
	go func() {
		defer close(s.doneCh)
		frames := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
		i := 0
		// 初始一行
		_, _ = fmt.Fprintf(s.out, "%s %c\r", s.msg, frames[i])
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				// 清理行尾
				_, _ = fmt.Fprintf(s.out, "%s ✔\n", s.msg)
				return
			case <-ticker.C:
				i = (i + 1) % len(frames)
				_, _ = fmt.Fprintf(s.out, "%s %c\r", s.msg, frames[i])
			}
		}
	}()
}

// Stop 停止 spinner.
func (s *Spinner) Stop() {
	close(s.stopCh)
	<-s.doneCh
}
