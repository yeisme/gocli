# 你好

```go
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/yeisme/gocli/pkg/style"
	"github.com/yeisme/gocli/pkg/utils/doc"
)

type DocOptions = doc.Options

func RunDoc(opts DocOptions, out io.Writer, args []string) error {
	// args 需要校验，至少有一个参数
	if len(args) < 1 {
		return fmt.Errorf("doc: at least one argument is required")
	}
	for _, file := range args {
		str, genErr := doc.GetDoc(opts, out, file)
		if genErr != nil {
			return fmt.Errorf("doc: failed to generate documentation for %q: %w", file, genErr)
		}
		style.RenderMarkdown(out, str, 0, "")
	}
	return nil
}
```

| t1  | t2  |
| --- | --- |
| 1   | 2   |
