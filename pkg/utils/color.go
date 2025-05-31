package utils

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// Predefined color functions
var (
	Red     = color.New(color.FgRed).SprintFunc()
	Green   = color.New(color.FgGreen).SprintFunc()
	Yellow  = color.New(color.FgYellow).SprintFunc()
	Blue    = color.New(color.FgBlue).SprintFunc()
	Magenta = color.New(color.FgMagenta).SprintFunc()
	Cyan    = color.New(color.FgCyan).SprintFunc()
	White   = color.New(color.FgWhite).SprintFunc()

	// Bold colors
	BoldRed     = color.New(color.FgRed, color.Bold).SprintFunc()
	BoldGreen   = color.New(color.FgGreen, color.Bold).SprintFunc()
	BoldYellow  = color.New(color.FgYellow, color.Bold).SprintFunc()
	BoldBlue    = color.New(color.FgBlue, color.Bold).SprintFunc()
	BoldMagenta = color.New(color.FgMagenta, color.Bold).SprintFunc()
	BoldCyan    = color.New(color.FgCyan, color.Bold).SprintFunc()
	BoldWhite   = color.New(color.FgWhite, color.Bold).SprintFunc()
)

// Initialize color settings
func init() {
	// Set color output based on global flags
	updateColorSettings()
}

// Update color settings based on global flags
func updateColorSettings() {
	isColor := IsColor()
	color.NoColor = !isColor
}

// Print functions with color support
func Printf(c func(a ...any) string, format string, args ...any) {
	if !IsQuiet() {
		updateColorSettings()
		fmt.Printf(c(format), args...)
	}
}

func Println(c func(a ...any) string, args ...any) {
	if !IsQuiet() {
		updateColorSettings()
		fmt.Println(c(fmt.Sprint(args...)))
	}
}

func Print(c func(a ...any) string, args ...any) {
	if !IsQuiet() {
		updateColorSettings()
		fmt.Print(c(fmt.Sprint(args...)))
	}
}

// Convenient output functions
func Success(msg string, args ...any) {
	Printf(BoldGreen, "[SUCCESS] "+msg+"\n", args...)
}

func Error(msg string, args ...any) {
	Printf(BoldRed, "[ERROR] "+msg+"\n", args...)
}

func Warning(msg string, args ...any) {
	Printf(BoldYellow, "[WARN] "+msg+"\n", args...)
}

func Info(msg string, args ...any) {
	Printf(BoldBlue, "[INFO] "+msg+"\n", args...)
}

func Debug(msg string, args ...any) {
	if IsVerbose() {
		Printf(Cyan, "[DEBUG] "+msg+"\n", args...)
	}
}

// Advanced output functions
func Header(msg string, args ...any) {
	if !IsQuiet() {
		updateColorSettings()
		line := "============================================"
		fmt.Println(BoldCyan(line))
		fmt.Printf(BoldWhite(msg+"\n"), args...)
		fmt.Println(BoldCyan(line))
	}
}

func SubHeader(msg string, args ...any) {
	if !IsQuiet() {
		updateColorSettings()
		fmt.Printf(BoldYellow(""+msg+":\n"), args...)
	}
}

// Progress and status output functions
func Progress(msg string, args ...any) {
	Printf(Yellow, "⏳ "+msg+"\n", args...)
}

func Complete(msg string, args ...any) {
	Printf(BoldGreen, "[✓] "+msg+"\n", args...)
}

// Error output to stderr
func ErrorToStderr(msg string, args ...any) {
	if !IsQuiet() {
		updateColorSettings()
		fmt.Fprintf(os.Stderr, BoldRed("[✗] "+msg+"\n"), args...)
	}
}

// List output functions
func ListItem(msg string, args ...any) {
	Printf(White, "• "+msg+"\n", args...)
}

func NumberedItem(num int, msg string, args ...any) {
	Printf(White, "%d. "+msg+"\n", append([]any{num}, args...)...)
}

// Colored box output
func Box(title string, content string) {
	if !IsQuiet() {
		updateColorSettings()
		width := 50
		border := "+" + fmt.Sprintf("%*s", width-2, "") + "+"
		for i := 1; i < width-1; i++ {
			border = border[:i] + "-" + border[i+1:]
		}

		fmt.Println(BoldCyan(border))
		titleLine := fmt.Sprintf("| %-*s |", width-4, title)
		fmt.Println(BoldWhite(titleLine))

		separator := "|" + fmt.Sprintf("%*s", width-2, "") + "|"
		for i := 1; i < width-1; i++ {
			separator = separator[:i] + "-" + separator[i+1:]
		}
		fmt.Println(BoldCyan(separator))

		// Output content, may need line break handling
		contentLines := []string{content} // Simplified handling, can be enhanced later
		for _, line := range contentLines {
			contentLine := fmt.Sprintf("| %-*s |", width-4, line)
			fmt.Println(White(contentLine))
		}

		fmt.Println(BoldCyan(border))
	}
}

// Disable/enable color functions
func DisableColor() {
	color.NoColor = true
}

func EnableColor() {
	color.NoColor = false
}

// Check if color is supported
func IsColorSupported() bool {
	return !color.NoColor
}
