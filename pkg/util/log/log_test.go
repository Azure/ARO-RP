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
		name  string
		f     *runtime.Frame
		want1 string
		want2 string
	}{
		{
			name:  "current function",
			f:     &currentFunc,
			want1: "log.TestRelativeFilePathPrettier()",
			want2: " pkg/util/log/log_test.go:11",
		},
		{
			name:  "empty",
			f:     &runtime.Frame{},
			want1: "()",
			want2: " :0",
		},
		{
			name: "install",
			f: &runtime.Frame{
				Function: "github.com/jim-minter/rp/pkg/install/install.installResources",
				File:     goPath + "src/github.com/jim-minter/rp/pkg/install/1-installresources.go",
				Line:     623,
			},
			want1: "install.installResources()",
			want2: " pkg/install/1-installresources.go:623",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got1, got2 := RelativeFilePathPrettier(tt.f)
			if got1 != tt.want1 {
				t.Errorf("RelativeFilePathPrettier() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("RelativeFilePathPrettier() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}
