package gitignore

import "fmt"

var tmplMap = map[string]string{
	"base-go": baseGoTmpl,
	"all":     all,
	"c-cpp":   cTmpl,
	"c-go":    cgoTmpl,
	"gocli":   gocliTmpl,
	"release": releaseTmpl,
}

var baseGoTmpl = `
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with "go test -c"
*.test

# Code coverage profiles and other test artifacts
*.out
coverage.*
*.coverprofile
profile.cov

# Dependency directories (remove the comment below to include it)
# vendor/

# Go workspace file
go.work
go.work.sum

# env file
.env

# Editor/IDE
.idea/
.vscode/
`

var all = `
*
`

var cTmpl = `
# Prerequisites
*.d

# Compiled Object files
*.slo
*.lo
*.o
*.obj

# Precompiled Headers
*.gch
*.pch

# Linker files
*.ilk

# Debugger Files
*.pdb

# Compiled Dynamic libraries
*.so
*.dylib
*.dll

# Compiled Static libraries
*.lai
*.la
*.a
*.lib

# Executables
*.exe
*.out
*.app

# debug information files
*.dwo

# IDE
.idea/
.vscode/
.vs/
`

var cgoTmpl = fmt.Sprintf("\n%s\n%s\n", baseGoTmpl, cTmpl)

var gocliTmpl = `
.gocli/
`

var releaseTmpl = `
dist/
*.zip
*.rar
*.7z
*.gz
*.tar
*.bz2
*.xz
`
