package log

import (
	"runtime"
	"strings"
	"testing"
)

func TestRelativeFilePathPrettier(t *testing.T) {
	pc := make([]uintptr, 1)
	runtime.Callers(1, pc)
	currentFrames := runtime.CallersFrames(pc)
	currentFunc, _ := currentFrames.Next()
	currentFunc.Line = 11 // so it's not too fragile
	goPath := strings.Split(currentFunc.File, "src/github.com")[0]
	tests := []struct {
		name         string
		f            *runtime.Frame
		wantFunction string
		wantFile     string
	}{
		{
			name:         "current function",
			f:            &currentFunc,
			wantFunction: "log.TestRelativeFilePathPrettier()",
			wantFile:     " pkg/util/log/log_test.go:11",
		},
		{
			name:         "empty",
			f:            &runtime.Frame{},
			wantFunction: "()",
			wantFile:     " :0",
		},
		{
			name: "install",
			f: &runtime.Frame{
				Function: "github.com/jim-minter/rp/pkg/install/install.installResources",
				File:     goPath + "src/github.com/jim-minter/rp/pkg/install/1-installresources.go",
				Line:     623,
			},
			wantFunction: "install.installResources()",
			wantFile:     " pkg/install/1-installresources.go:623",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			function, file := RelativeFilePathPrettier(tt.f)
			if function != tt.wantFunction {
				t.Error(function)
			}
			if file != tt.wantFile {
				t.Error(file)
			}
		})
	}
}
