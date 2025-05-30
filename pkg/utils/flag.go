package utils

var (
	verbose    bool
	useColor   bool
	quiet      bool
	user       bool
	configfile string
)

func SetConfigFile(cf string) {
	configfile = cf
}

func GetConfigFile() string {
	return configfile
}

func SetGlobalFlags(v, c, q, u bool) {
	verbose = v
	useColor = c
	quiet = q
	user = u
	if verbose && quiet {
		Error("Cannot use both --verbose and --quiet flags together.")
	}
}
func GetGlobalFlags() (bool, bool, bool, bool) {
	return verbose, useColor, quiet, user
}

func IsVerbose() bool {
	return verbose
}
func IsColor() bool {
	return useColor
}
func IsQuiet() bool {
	return quiet
}
func IsUser() bool {
	return user
}
