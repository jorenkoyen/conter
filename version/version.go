package version

import (
	"runtime"
	"strings"
)

var Version = "0.0.0"
var GoVersion = strings.TrimPrefix(runtime.Version(), "go")
