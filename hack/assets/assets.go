package main

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/shurcooL/vfsgen"
)

var allowedPaths = []*regexp.Regexp{
	regexp.MustCompile(`^/bootstrap$`),
	regexp.MustCompile(`^/bootstrap/files($|/)`),
	regexp.MustCompile(`^/bootstrap/systemd($|/)`),
	regexp.MustCompile(`^/manifests$`),
	regexp.MustCompile(`^/manifests/bootkube($|/)`),
	regexp.MustCompile(`^/manifests/openshift($|/)`),
	regexp.MustCompile(`^/rhcos.json$`),
}

type fileSystem struct {
	http.FileSystem
}

func (fs fileSystem) Open(name string) (http.File, error) {
	f, err := fs.FileSystem.Open(name)
	return file{f, name}, err
}

type file struct {
	http.File
	name string
}

func (f file) Readdir(count int) ([]os.FileInfo, error) {
	fis, err := f.File.Readdir(count)
	out := make([]os.FileInfo, 0, len(fis))
	for _, fi := range fis {
		for _, allowedPath := range allowedPaths {
			if allowedPath.MatchString(filepath.Join(f.name, fi.Name())) {
				out = append(out, fi)
				break
			}
		}
	}
	return out, err
}

func (f file) Stat() (os.FileInfo, error) {
	fi, err := f.File.Stat()
	return fileInfo{fi}, err
}

type fileInfo struct {
	os.FileInfo
}

func (fi fileInfo) ModTime() time.Time { return time.Time{} }

func main() {
	err := vfsgen.Generate(fileSystem{http.Dir("../../vendor/github.com/openshift/installer/data/data")}, vfsgen.Options{
		Filename:     "../../pkg/deploy/assets_vfsdata.go",
		PackageName:  "deploy",
		VariableName: "Assets",
	})
	if err != nil {
		panic(err)
	}
}
