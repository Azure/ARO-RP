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
	file := strings.TrimPrefix(f.File, repopath)
	function := f.Function[strings.LastIndexByte(f.Function, '/')+1:]
	return fmt.Sprintf("%s()", function), fmt.Sprintf(" %s:%d", file, f.Line)
}
