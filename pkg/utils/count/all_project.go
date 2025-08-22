package count

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/yeisme/gocli/pkg/models"
	"github.com/yeisme/gocli/pkg/utils/gitignore"
)

// ProjectCounter 是 CountProject 接口的实现
// 它通过组合（Composition）不同的计数器接口（用于单文件和Go语言特定细节），
// 实现了对整个代码项目的统计功能这种设计提高了模块化和可测试性
type ProjectCounter struct {
	FileCounter File   // 用于统计单个文件的基本信息（如代码行、注释行等）的接口
	GoCounter   GoFile // 用于统计 Go 语言文件特定细节（如函数数量、结构体数量）的接口
}

// CountAllFiles 遍历指定根目录（项目），根据提供的选项（Options）筛选文件，
// 并为每个符合条件的文件计算统计信息, 它会并发处理文件以提高性能
//
//	ctx: 用于控制函数执行的上下文，例如可以用于超时或取消操作
//	root: 要统计的项目的根目录路径
//	opts: 包含各种统计选项，如包含/排除规则、是否遵循符号链接等
//	返回值: 一个包含所有已处理文件信息的切片，以及遇到的第一个非文件大小限制的错误
func (p *ProjectCounter) CountAllFiles(ctx context.Context, root string, opts Options) ([]models.FileInfo, error) {
	// 确保内部的计数器都已初始化，防止空指针异常
	p = ensureCounters(p)

	// 如果选项要求，加载项目根目录下的 .gitignore 文件
	gi := loadGitIgnore(root, opts.RespectGitignore)

	// 步骤1: 收集所有需要处理的文件路径
	// 这个阶段会遍历目录，并根据 .gitignore、include/exclude 规则、文件大小等进行过滤，并且过滤一些常见的目录 .git
	filesToProcess, err := collectFiles(ctx, root, opts, gi)
	if err != nil {
		return nil, err
	}

	// 步骤2: 准备并发处理根据用户设置或CPU核心数确定并发的 worker 数量
	conc := prepareConcurrency(opts.Concurrency)

	// 步骤3: 并发处理所有收集到的文件，并收集结果
	results, firstErr := processFilesConcurrently(ctx, p, root, filesToProcess, opts, conc)
	// 如果处理过程中发生错误，并且没有成功处理任何文件，则返回错误
	// 否则，即使有错误，也可能返回部分成功的结果
	if firstErr != nil && len(results) == 0 {
		return nil, firstErr
	}
	return results, nil
}

// CountProjectSummary 在 CountAllFiles 的基础上，对所有文件的统计结果进行聚合
// 它将结果按编程语言分组，并计算整个项目的总计信息
//
//	ctx: 用于控制函数执行的上下文
//	root: 要统计的项目的根目录路径
//	opts: 统计选项
//
// 返回值: 一个包含详细聚合分析结果的指针，或者在获取文件列表时发生的错误
func (p *ProjectCounter) CountProjectSummary(ctx context.Context, root string, opts Options) (*models.AnalysisResult, error) {
	// 首先，获取所有独立文件的统计信息
	files, err := p.CountAllFiles(ctx, root, opts)
	if err != nil {
		return nil, err
	}
	// 然后，将这些独立的文件信息聚合成一个总的分析报告
	return aggregateAnalysis(files, opts), nil
}

// -----------------------------------------------------------------------------
// 工具函数 (Utility Functions)
// -----------------------------------------------------------------------------

// matchesAny 检查给定的相对路径 `relPath` 是否匹配 `patterns` 中的任意一个模式
// 这个函数是 include/exclude 功能的核心
// 它支持两种匹配方式:
//  1. `filepath.Match`:进行标准的 shell glob 模式匹配（如 `*.go`, `cmd/*`）
//  2. `strings.Contains`:作为备用，进行简单的子串包含匹配
//
// 这种双重检查提供了灵活性
func matchesAny(relPath string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// 优先使用更精确的 glob 模式匹配
		if ok, _ := filepath.Match(p, relPath); ok {
			return true
		}
		// 如果 glob 匹配失败，退而求其次，检查是否为子串
		if strings.Contains(relPath, p) {
			return true
		}
	}
	return false
}

// isSymlink 检查一个目录条目 `fs.DirEntry` 是否是符号链接
func isSymlink(d fs.DirEntry) bool {
	// 通过位掩码检查文件模式是否包含符号链接的标志位
	return d.Type()&fs.ModeSymlink != 0
}

// isSizeLimitError 用于识别因文件过大而被跳过时可能产生的错误
// 注意:在当前的实现中，我们在遍历（WalkDir）时已经通过 `overSize` 函数提前过滤了过大的文件，
// 所以这个函数主要用于防御性编程或兼容未来的修改
func isSizeLimitError(err error) bool {
	if err == nil {
		return false
	}
	// Go 标准库中没有一个特定的错误类型来表示“文件过大"
	// 这是一个尝试性的检查，但目前没有稳定的方式来识别它，因此通常返回 false
	var pathErr *fs.PathError
	return errors.As(err, &pathErr) && strings.Contains(strings.ToLower(pathErr.Error()), "size")
}

// `var _ CountProject = (*ProjectCounter)(nil)` 是一个编译时断言
// 它确保 *ProjectCounter 类型确实实现了 CountProject 接口
// 如果接口定义发生变化而 ProjectCounter 没有相应更新，编译将会失败，
// 从而可以及早发现问题
var _ Project = (*ProjectCounter)(nil)

// ensureCounters 确保 ProjectCounter 中的计数器字段不为 nil
// 如果外部创建 ProjectCounter 时未提供具体的计数器实现，
// 这个函数会为其分配置默认的实现（SingleFileCounter 和 GoDetailsCounter）
// 这样可以避免在后续调用中出现空指针引用
func ensureCounters(p *ProjectCounter) *ProjectCounter {
	if p == nil {
		p = &ProjectCounter{}
	}
	if p.FileCounter == nil {
		p.FileCounter = &SingleFileCounter{}
	}
	if p.GoCounter == nil {
		p.GoCounter = &GoDetailsCounter{}
	}
	return p
}

// loadGitIgnore 尝试从指定的项目根目录加载 `.gitignore` 文件
// 如果 `respect` 参数为 false，或者加载失败，它会返回一个空的或 nil 的 gitignore 处理器，
// 这样后续的检查 `gi.IsIgnored()` 将始终返回 false，相当于禁用了 gitignore 功能
func loadGitIgnore(root string, respect bool) *gitignore.GitIgnore {
	if !respect {
		return nil
	}
	gi, err := gitignore.LoadGitIgnoreFromDir(root)
	if err != nil {
		// 即使加载失败，也返回一个空的实例，以避免后续代码中的 nil 检查
		return &gitignore.GitIgnore{}
	}
	return gi
}

// collectFiles 使用 `filepath.WalkDir` 递归遍历项目目录，收集所有符合条件的文件路径
// 这是文件发现和过滤的主要逻辑所在
//
//	ctx: 用于取消遍历过程
//	root: 遍历的起始目录
//	opts: 包含过滤规则的选项
//	gi: 已加载的 gitignore 规则处理器
func collectFiles(ctx context.Context, root string, opts Options, gi *gitignore.GitIgnore) ([]string, error) {
	// 预分配切片容量，提高性能256 是一个合理的初始猜测值
	files := make([]string, 0, 256)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		// 首先处理遍历过程中可能发生的 I/O 错误
		if walkErr != nil {
			return walkErr
		}
		// 检查上下文是否已被取消如果被取消，则立即停止遍历
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// --- 目录处理逻辑 ---
		if d.IsDir() {
			// 如果是根目录本身，直接继续
			if path == root {
				return nil
			}
			// 将目录路径转换为相对于 root 的、使用 `/`作为分隔符的路径
			relSlash := toRelSlash(root, path)
			// 判断是否应该跳过整个目录
			if shouldSkipDir(relSlash, opts, gi) {
				// 返回 filepath.SkipDir 会告诉 WalkDir 不要进入这个目录
				return filepath.SkipDir
			}
			return nil
		}

		// --- 文件处理逻辑 ---
		relSlash := toRelSlash(root, path)
		// 判断是否应该包含这个文件
		if !shouldIncludeFile(relSlash, opts, gi) {
			return nil
		}

		// 处理符号链接
		if isSymlink(d) {
			// 如果选项配置为不跟随符号链接，则跳过
			if !opts.FollowSymlinks {
				return nil
			}
			// 如果跟随，依然要检查链接指向的文件大小是否超限
			if overSize(path, opts.MaxFileSizeBytes) {
				return nil
			}
			files = append(files, path)
			return nil
		}

		// 对于普通文件，检查是否超过大小限制
		if overSize(path, opts.MaxFileSizeBytes) {
			return nil
		}

		// 如果所有检查都通过，将文件路径添加到待处理列表中
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// toRelSlash 将绝对路径 `path` 转换为相对于 `root` 的路径，并确保路径分隔符为 `/`
// 这对于跨平台（Windows/Linux/macOS）保持路径模式匹配的一致性非常重要
func toRelSlash(root, path string) string {
	rel, _ := filepath.Rel(root, path)
	return filepath.ToSlash(rel)
}

// shouldSkipDir 判断是否应该跳过一个目录
// 跳过的条件:
//  1. 目录被 `.gitignore` 规则匹配
//  2. 没有设置 `Include` 规则，但目录匹配了 `Exclude` 规则
func shouldSkipDir(relSlash string, opts Options, gi *gitignore.GitIgnore) bool {
	// 默认忽略任意层级的 .git 目录（例如 .git, foo/.git, a/b/.git）
	if relSlash == ".git" || strings.HasSuffix(relSlash, "/.git") || strings.Contains(relSlash, "/.git/") {
		return true
	}
	if gi != nil && gi.IsIgnored(relSlash) {
		return true
	}
	// 当 Include 列表为空时，Exclude 规则才对目录生效
	// 这是为了避免排除一个目录，但其子文件可能被 Include 规则包含的情况
	if len(opts.Include) == 0 {
		// 先做普通的匹配（glob / contains）
		if matchesAny(relSlash, opts.Exclude) {
			return true
		}
		// 另外支持像 `pkg/*`、`pkg/`、`pkg` 这样的排除模式匹配目录及其子项
		for _, raw := range opts.Exclude {
			p := strings.TrimSpace(raw)
			if p == "" {
				continue
			}
			// 规范化为使用 `/` 的形式，并去掉前导 `./` 或 `.\\`
			p = strings.ReplaceAll(p, "\\", "/")
			if after, ok := strings.CutPrefix(p, "./"); ok {
				p = after
			}
			if after, ok := strings.CutPrefix(p, ".\\"); ok {
				p = after
			}
			// 如果模式以 `/*` 结尾，去掉通配部分然后比较前缀
			if strings.HasSuffix(p, "/*") {
				prefix := strings.TrimSuffix(p, "/*")
				if prefix == relSlash || strings.HasPrefix(relSlash, prefix+"/") {
					return true
				}
				continue
			}
			// 如果模式以 `/` 结尾，认为是目录，直接比较前缀
			if strings.HasSuffix(p, "/") {
				prefix := strings.TrimSuffix(p, "/")
				if prefix == relSlash || strings.HasPrefix(relSlash, prefix+"/") {
					return true
				}
				continue
			}
			// 直接的前缀匹配（例如用户传入 pkg 或 pkg/some）也应当生效
			if p == relSlash || strings.HasPrefix(p, relSlash+"/") || strings.HasPrefix(relSlash, p+"/") {
				return true
			}
		}
	}
	return false
}

// shouldIncludeFile 判断是否应该包含一个文件用于统计
// 包含的逻辑优先级:
//  1. 如果被 `.gitignore` 忽略，则不包含
//  2. 如果 `Include` 列表不为空，则只有匹配 `Include` 列表的文件才被包含
//  3. 如果 `Include` 列表为空，则检查文件是否匹配 `Exclude` 列表，匹配则不包含
//  4. 如果以上条件都不满足，则默认包含
func shouldIncludeFile(relSlash string, opts Options, gi *gitignore.GitIgnore) bool {
	if gi != nil && gi.IsIgnored(relSlash) {
		return false
	}
	if includeMatches(relSlash, opts.Include) {
		return true
	}
	if len(opts.Include) > 0 {
		return false
	}
	if excludeMatches(relSlash, opts.Exclude) {
		return false
	}
	return true
}

// normalizePattern 将用户传入的模式标准化为使用 `/` 的路径形式并去掉前导的 `./` 或 `.\`
func normalizePattern(raw string) string {
	p := strings.TrimSpace(raw)
	if p == "" {
		return ""
	}
	p = strings.ReplaceAll(p, "\\", "/")
	if after, ok := strings.CutPrefix(p, "./"); ok {
		p = after
	}
	if after, ok := strings.CutPrefix(p, ".\\"); ok {
		p = after
	}
	return p
}

// includeMatches 检查相对路径是否匹配任意 include 模式
func includeMatches(rel string, include []string) bool {
	if len(include) == 0 {
		return false
	}
	for _, raw := range include {
		p := normalizePattern(raw)
		if p == "" {
			continue
		}
		if ok, _ := filepath.Match(p, rel); ok {
			return true
		}
		if strings.HasSuffix(p, "/*") {
			prefix := strings.TrimSuffix(p, "/*")
			if prefix == rel || strings.HasPrefix(rel, prefix+"/") {
				return true
			}
			continue
		}
		if strings.HasSuffix(p, "/") {
			prefix := strings.TrimSuffix(p, "/")
			if prefix == rel || strings.HasPrefix(rel, prefix+"/") {
				return true
			}
			continue
		}
		if strings.Contains(rel, p) || strings.HasPrefix(rel, p+"/") || p == rel {
			return true
		}
	}
	return false
}

// excludeMatches 检查相对路径是否匹配任意 exclude 模式
func excludeMatches(rel string, exclude []string) bool {
	for _, raw := range exclude {
		p := normalizePattern(raw)
		if p == "" {
			continue
		}
		if ok, _ := filepath.Match(p, rel); ok {
			return true
		}
		if strings.HasSuffix(p, "/*") {
			prefix := strings.TrimSuffix(p, "/*")
			if prefix == rel || strings.HasPrefix(rel, prefix+"/") {
				return true
			}
			continue
		}
		if strings.HasSuffix(p, "/") {
			prefix := strings.TrimSuffix(p, "/")
			if prefix == rel || strings.HasPrefix(rel, prefix+"/") {
				return true
			}
			continue
		}
		if strings.Contains(rel, p) ||
			strings.HasPrefix(p, rel+"/") ||
			strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
}

// overSize 检查文件大小是否超过给定的限制 `limit`
// 如果 limit <= 0，则表示没有大小限制
func overSize(path string, limit int64) bool {
	if limit <= 0 {
		return false
	}
	if st, err := os.Stat(path); err == nil {
		return st.Size() > limit
	}
	// 如果获取文件状态失败，保守地认为它没有超大，让后续处理步骤去报告这个错误
	return false
}

// prepareConcurrency 确定用于处理文件的并发 worker 数量
// 如果用户指定了正数的并发数 `c`，则使用该值
// 否则，默认使用机器的 CPU 核心数，但至少为 1
func prepareConcurrency(c int) int {
	if c > 0 {
		return c
	}
	return max(runtime.NumCPU(), 1)
}

// processFilesConcurrently 使用一个 worker pool 模型来并发处理文件列表
// conc: worker 的数量
//
// 工作流程:
//  1. 创建两个 channel:`inCh` 用于分发文件路径任务，`outCh` 用于收集处理结果
//  2. 启动 `conc` 个 `worker` goroutine每个 worker 从 `inCh` 读取任务，处理后将结果写入 `outCh`
//  3. 启动一个 goroutine，负责将 `files` 列表中的所有路径发送到 `inCh`，发送完毕后关闭 `inCh`
//  4. 主 goroutine 从 `outCh` 读取所有结果，直到 `outCh` 被关闭
//  5. 使用 `sync.WaitGroup` 确保所有 worker 都完成后再关闭 `outCh`
func processFilesConcurrently(
	ctx context.Context,
	p *ProjectCounter,
	root string,
	files []string,
	opts Options,
	conc int,
) ([]models.FileInfo, error) {
	// 定义一个内部类型，用于在 channel 中传递结果或错误
	type item struct {
		info models.FileInfo
		err  error
	}

	inCh := make(chan string)
	outCh := make(chan item)
	var wg sync.WaitGroup

	// 定义 worker 函数
	worker := func() {
		defer wg.Done()
		for path := range inCh {
			info, err := processFile(ctx, p, root, path, opts)
			if err != nil {
				outCh <- item{err: err}
				continue
			}
			outCh <- item{info: info}
		}
	}

	// 启动指定数量的 worker
	wg.Add(conc)
	for range conc {
		go worker()
	}

	// 启动任务分发器
	go func() {
		// 在这个 goroutine 退出前，要确保关闭 outCh
		// 这需要等待所有 worker 都完成（wg.Wait()）
		defer close(outCh)
		for _, f := range files {
			// 在发送任务前检查上下文是否已取消
			if ctx.Err() != nil {
				break
			}
			inCh <- f
		}
		// 所有任务都已发送，关闭 inCh，worker 会在读取完 channel 后自动退出
		close(inCh)
		// 等待所有 worker 执行完毕
		wg.Wait()
	}()

	// 在主 goroutine 中收集结果
	results := make([]models.FileInfo, 0, len(files))
	var firstErr error
	for it := range outCh {
		if it.err != nil {
			// 忽略因文件过大而产生的错误，但记录遇到的第一个其他类型的错误
			if !isSizeLimitError(it.err) && firstErr == nil {
				firstErr = it.err
			}
			continue
		}
		results = append(results, it.info)
	}

	return results, firstErr
}

// processFile 处理单个文件的统计任务
// 它调用相应的计数器，并根据选项处理特定语言的细节
func processFile(ctx context.Context, p *ProjectCounter, root, path string, opts Options) (models.FileInfo, error) {
	// 在处理前再次检查上下文状态
	if ctx.Err() != nil {
		return models.FileInfo{}, ctx.Err()
	}

	// 调用通用的文件计数器
	fi, err := p.FileCounter.CountSingleFile(ctx, path, opts)
	if err != nil {
		return models.FileInfo{}, err
	}

	// 将文件的绝对路径转换为相对路径，便于显示
	if rel, rerr := filepath.Rel(root, path); rerr == nil {
		fi.Path = rel
	}

	// 如果文件是 Go 文件，并且选项要求分析特定语言细节
	if opts.WithLanguageSpecific && fi.Language == "Go" {
		// 调用 Go 语言专用的计数器
		if details, derr := p.GoCounter.CountGoDetails(ctx, path); derr == nil && details != nil {
			// 根据选项决定是否包含函数和结构体的计数
			if !opts.WithFunctions {
				details.Functions = 0
			}
			if !opts.WithStructs {
				details.Structs = 0
			}
			fi.LanguageSpecific = details
		}
	}

	return *fi, nil
}

// aggregateAnalysis 将一组独立的文件统计信息 (`files`) 聚合成一个最终的分析结果
// 结果会按语言进行分组，并计算总计
func aggregateAnalysis(files []models.FileInfo, opts Options) *models.AnalysisResult {
	res := &models.AnalysisResult{
		Total:     models.LanguageStats{},
		Languages: make(map[string]*models.LanguageStats),
	}
	for _, f := range files {
		lang := f.Language
		// 如果语言无法识别，归类为 "Unknown"
		if lang == "" {
			lang = "Unknown"
		}

		// 获取或创建该语言的统计对象
		ls, ok := res.Languages[lang]
		if !ok {
			ls = &models.LanguageStats{}
			res.Languages[lang] = ls
		}

		// 累加该语言的统计数据
		ls.FileCount++
		ls.Stats.Code += f.Stats.Code
		ls.Stats.Comments += f.Stats.Comments
		ls.Stats.Blanks += f.Stats.Blanks
		// 若语言特定信息中包含函数/结构体并且开启统计，则聚合
		if opts.WithFunctions || opts.WithStructs {
			if gd, ok := f.LanguageSpecific.(*models.GoDetails); ok && gd != nil {
				if opts.WithFunctions {
					ls.Functions += gd.Functions
				}
				if opts.WithStructs {
					ls.Structs += gd.Structs
				}
			}
		}
		// 如果选项要求，将文件的详细信息添加到语言分组中
		if opts.WithLanguageDetails {
			ls.Files = append(ls.Files, f)
		}

		// 同时累加到项目总计中
		res.Total.FileCount++
		res.Total.Stats.Code += f.Stats.Code
		res.Total.Stats.Comments += f.Stats.Comments
		res.Total.Stats.Blanks += f.Stats.Blanks
		if opts.WithFunctions || opts.WithStructs {
			if gd, ok := f.LanguageSpecific.(*models.GoDetails); ok && gd != nil {
				if opts.WithFunctions {
					res.Total.Functions += gd.Functions
				}
				if opts.WithStructs {
					res.Total.Structs += gd.Structs
				}
			}
		}

		// 如果顶层需要文件列表（不分语言），收集
		if opts.WithFileDetails {
			res.Files = append(res.Files, f)
		}
	}
	return res
}
