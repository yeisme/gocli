// Package version provides version information for the gocli application.
// It includes build-time information such as version, git commit, build date, etc.
package version

import (
	"fmt"
	"runtime"
	"time"
)

var (
	// Version is the current version of the application
	Version = "dev"
	// GitCommit is the git commit hash
	GitCommit = "unknown"
	// BuildDate is when the binary was built
	BuildDate = "unknown"
	// GoVersion is the Go version used to build the binary
	GoVersion = runtime.Version()
	// Platform is the target platform
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	// Modified indicates if the source tree was modified (string: "true" or "false")
	Modified = "false"
	// ModSum is the module checksum
	ModSum = "unknown"
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
	Modified  string `json:"modified"`
	ModSum    string `json:"mod_sum"`
}

// GetVersion returns the version information
func GetVersion() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		Platform:  Platform,
		Modified:  Modified,
		ModSum:    ModSum,
	}
}

// GetVersionString returns a formatted version string similar to gh and golangci-lint
func GetVersionString() string {
	info := GetVersion()

	modifiedStr := fmt.Sprintf(", modified: %s", info.Modified)

	// Format similar to golangci-lint
	return fmt.Sprintf("gocli has version %s built with %s from %s (%s%s, mod sum: \"%s\") on %s",
		info.Version,
		info.GoVersion,
		info.GitCommit,
		info.Platform,
		modifiedStr,
		info.ModSum,
		info.BuildDate,
	)
}

// GetShortVersionString returns a short version string similar to gh
func GetShortVersionString() string {
	info := GetVersion()

	// Parse build date for URL formatting
	buildTime, err := time.Parse(time.RFC3339, info.BuildDate)
	var dateStr string
	if err != nil {
		dateStr = info.BuildDate
	} else {
		dateStr = buildTime.Format("2006-01-02")
	}

	// Format similar to GitHub CLI
	return fmt.Sprintf("gocli version %s (%s)\nhttps://github.com/yeisme/gocli/releases/tag/v%s",
		info.Version,
		dateStr,
		info.Version,
	)
}
