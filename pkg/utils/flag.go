package utils

var (
	verbose  bool
	useColor bool
	quiet    bool
)

func SetGlobalFlags(v, c, q bool) {
	verbose = v
	useColor = c
	quiet = q
}
func GetGlobalFlags() (bool, bool, bool) {
	return verbose, useColor, quiet
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
