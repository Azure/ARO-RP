package log

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	_, thisfile, _, _ = runtime.Caller(0)
	repopath          = strings.Replace(thisfile, "pkg/util/log/log.go", "", -1)
)

// RelativeFilePathPrettier changes absolute paths with relative paths
func RelativeFilePathPrettier(f *runtime.Frame) (string, string) {
	filename := strings.Replace(f.File, repopath, "", -1)
	funcname := strings.Replace(f.Function, "github.com/jim-minter/rp/", "", -1)
	return fmt.Sprintf("%s()", funcname), fmt.Sprintf("%s:%d", filename, f.Line)
}
