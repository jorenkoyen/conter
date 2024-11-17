package version

import (
	"fmt"
	"runtime"
	"strings"
)

var Version = "0.0.0"
var GoVersion = strings.TrimPrefix(runtime.Version(), "go")

func UserAgent() string {
	return fmt.Sprintf("conter/%s go/%s (%s %s)", Version, GoVersion, runtime.GOARCH, runtime.GOOS)
}
