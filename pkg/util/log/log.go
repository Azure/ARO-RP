package log

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	_, thisfile, _, _ = runtime.Caller(0)
	repopath          = strings.Replace(thisfile, "pkg/util/log/log.go", "", -1)
)

// GetLogger returns a consistently configured log entry
func GetLogger() *logrus.Entry {
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		CallerPrettyfier: RelativeFilePathPrettier,
	})

	return logrus.NewEntry(logrus.StandardLogger())
}

// RelativeFilePathPrettier changes absolute paths with relative paths
func RelativeFilePathPrettier(f *runtime.Frame) (string, string) {
	file := strings.TrimPrefix(f.File, repopath)
	function := f.Function[strings.LastIndexByte(f.Function, '/')+1:]
	return fmt.Sprintf("%s()", function), fmt.Sprintf(" %s:%d", file, f.Line)
}
