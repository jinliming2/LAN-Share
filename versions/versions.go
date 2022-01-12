package versions

import (
	"fmt"
	"runtime"
)

var (
	// PROGRAM is the name of this software
	PROGRAM = "LAN Share"
	// VERSION should be replaced at compile
	VERSION = "UNKNOWN"
	// BUILDHASH should be replaced at compile
	BUILDHASH = ""
)

// PrintVersion print version information
func PrintVersion() {
	fmt.Printf("%s/%s %s %s/%s\n", PROGRAM, VERSION, BUILDHASH, runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Build with Go %s (%s)\n", runtime.Version(), runtime.Compiler)
}
