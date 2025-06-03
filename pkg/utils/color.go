package utils

import (
	"fmt"
	"os"
	"strings"

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

	// Italic colors
	ItalicRed     = color.New(color.FgRed, color.Italic).SprintFunc()
	ItalicGreen   = color.New(color.FgGreen, color.Italic).SprintFunc()
	ItalicYellow  = color.New(color.FgYellow, color.Italic).SprintFunc()
	ItalicBlue    = color.New(color.FgBlue, color.Italic).SprintFunc()
	ItalicMagenta = color.New(color.FgMagenta, color.Italic).SprintFunc()
	ItalicCyan    = color.New(color.FgCyan, color.Italic).SprintFunc()
	ItalicWhite   = color.New(color.FgWhite, color.Italic).SprintFunc()

	// Underlined colors
	UnderlineRed     = color.New(color.FgRed, color.Underline).SprintFunc()
	UnderlineGreen   = color.New(color.FgGreen, color.Underline).SprintFunc()
	UnderlineYellow  = color.New(color.FgYellow, color.Underline).SprintFunc()
	UnderlineBlue    = color.New(color.FgBlue, color.Underline).SprintFunc()
	UnderlineMagenta = color.New(color.FgMagenta, color.Underline).SprintFunc()
	UnderlineCyan    = color.New(color.FgCyan, color.Underline).SprintFunc()
	UnderlineWhite   = color.New(color.FgWhite, color.Underline).SprintFunc()

	// Combined styles
	BoldItalicGreen  = color.New(color.FgGreen, color.Bold, color.Italic).SprintFunc()
	BoldItalicRed    = color.New(color.FgRed, color.Bold, color.Italic).SprintFunc()
	BoldItalicYellow = color.New(color.FgYellow, color.Bold, color.Italic).SprintFunc()
	BoldItalicCyan   = color.New(color.FgCyan, color.Bold, color.Italic).SprintFunc()

	// Faint colors for subtle text
	FaintWhite   = color.New(color.FgWhite, color.Faint).SprintFunc()
	FaintCyan    = color.New(color.FgCyan, color.Faint).SprintFunc()
	FaintYellow  = color.New(color.FgYellow, color.Faint).SprintFunc()
	FaintMagenta = color.New(color.FgMagenta, color.Faint).SprintFunc()
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
	Printf(BoldItalicGreen, "[SUCCESS] "+msg+"\n", args...)
}

func Error(msg string, args ...any) {
	Printf(BoldRed, "[ERROR] "+msg+"\n", args...)
}

func Warning(msg string, args ...any) {
	Printf(ItalicYellow, "[WARNING] "+msg+"\n", args...)
}

func Info(msg string, args ...any) {
	Printf(Blue, "[INFO] "+msg+"\n", args...)
}

func Debug(msg string, args ...any) {
	if IsVerbose() {
		Printf(White, "[DEBUG] "+msg+"\n", args...)
	}
}

// Advanced output functions
func Header(msg string, args ...any) {
	if !IsQuiet() {
		updateColorSettings()
		width := 50

		// Top border
		topBorder := "╔" + strings.Repeat("═", width-2) + "╗"
		fmt.Println(BoldCyan(topBorder))

		// Title
		title := fmt.Sprintf(msg, args...)
		titleLine := fmt.Sprintf("║ %-*s ║", width-4, title)
		fmt.Println(BoldItalicCyan(titleLine))

		// Bottom border
		bottomBorder := "╚" + strings.Repeat("═", width-2) + "╝"
		fmt.Println(BoldCyan(bottomBorder))
	}
}

func SubHeader(msg string, args ...any) {
	if !IsQuiet() {
		updateColorSettings()
		fmt.Printf(UnderlineYellow("> "+msg+":\n"), args...)
	}
}

// Progress and status output functions
func Progress(msg string, args ...any) {
	Printf(ItalicYellow, "[PROGRESS] "+msg+"\n", args...)
}

func Complete(msg string, args ...any) {
	Printf(FaintWhite, "[COMPLETE] "+msg+"\n", args...)
}

// Error output to stderr
func ErrorToStderr(msg string, args ...any) {
	if !IsQuiet() {
		updateColorSettings()
		fmt.Fprintf(os.Stderr, BoldItalicRed("[FATAL] "+msg+"\n"), args...)
	}
}

// List output functions
func ListItem(msg string, args ...any) {
	Printf(White, " • "+msg+"\n", args...)
}

func NumberedItem(num int, msg string, args ...any) {
	Printf(FaintMagenta, "%d. ", num)
	Printf(White, msg+"\n", args...)
}

// Colored box output
func Box(title string, content string, width int) {
	if width == 0 {
		width = len(title) + 10
	}
	if !IsQuiet() {
		updateColorSettings()

		// Top border
		topBorder := "┌" + strings.Repeat("─", width-2) + "┐"
		fmt.Println(ItalicWhite(topBorder))

		// Title
		titleLine := fmt.Sprintf("│ %-*s │", width-4, title)
		fmt.Println(ItalicWhite(titleLine))

		// Separator
		separator := "├" + strings.Repeat("─", width-2) + "┤"
		fmt.Println(ItalicWhite(separator))

		// Content - split by newlines
		contentLines := strings.Split(content, "\n")
		for _, line := range contentLines {
			// Handle long lines by truncating if necessary
			if len(line) > width-4 {
				line = line[:width-7] + "..."
			}
			contentLine := fmt.Sprintf("│ %-*s │", width-4, line)
			fmt.Println(ItalicWhite(contentLine))
		}

		// Bottom border
		bottomBorder := "└" + strings.Repeat("─", width-2) + "┘"
		fmt.Println(ItalicWhite(bottomBorder))
	}
}

// Disable/enable color functions
func DisableColor() {
	color.NoColor = true
}

func EnableColor() {
	color.NoColor = false
}

// Check if color is supported/ Check if color is supported
func IsColorSupported() bool {
	return !color.NoColor
}
