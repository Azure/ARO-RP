// Code generated for package machinehealthcheck by go-bindata DO NOT EDIT. (@generated)
// sources:
// staticresources/machinehealthcheck.yaml
package machinehealthcheck

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _machinehealthcheckYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x9c\x8f\x31\x4f\xc3\x30\x10\x85\xf7\xfc\x8a\x93\xf7\x96\x56\x6c\x5e\x2b\x21\x06\x58\x90\xca\x7e\x75\x0e\xd9\x8a\x7d\x67\xf9\x2e\x94\xfe\x7b\x94\x98\x66\x62\x81\xcd\xb2\xbf\xf7\xbd\x67\xac\xe9\x9d\x9a\x26\x61\x0f\x05\x43\x4c\x4c\x7b\xa9\xc4\x1a\xd3\x87\xed\x93\x3c\x7c\x1e\x2f\x64\x78\x1c\xa6\xc4\xa3\x87\xd7\x8e\x3c\x13\x66\x8b\xa7\x48\x61\x1a\x0a\x19\x8e\x68\xe8\x07\x00\xc6\x42\x1e\xb0\xc9\xee\xc7\x15\x57\x30\xac\x60\x7f\xd6\x8a\x81\x3c\x6c\x1d\x77\x72\x87\x35\x0d\x5a\x29\x2c\x1e\xa5\x4c\xc1\xa4\x2d\x67\x80\x82\x16\xe2\x0b\x5e\x28\x6b\xbf\x80\xdf\xa7\x86\x3c\xab\x51\x5b\x4c\x9b\xb5\x49\x26\x0f\x57\x69\x13\x35\xf8\x6b\xd8\x6e\x75\x0b\x0f\x00\x33\xf7\xef\xdc\x4e\xc2\x63\xb2\x24\xbc\xee\xd9\x41\xe7\x00\xc0\xbd\x11\x8e\x37\xb7\xf6\x58\x2a\x24\xb3\x79\x70\x8f\x87\x83\xba\x5e\xae\x86\x36\xab\x07\xf7\x84\x59\xc9\xfd\x37\x7d\xe6\x89\xe5\xca\x0b\x5a\xf0\xeb\x7c\xdf\xe5\xc1\x1d\xdd\x77\x00\x00\x00\xff\xff\x04\x79\x27\x4a\xd2\x01\x00\x00")

func machinehealthcheckYamlBytes() ([]byte, error) {
	return bindataRead(
		_machinehealthcheckYaml,
		"machinehealthcheck.yaml",
	)
}

func machinehealthcheckYaml() (*asset, error) {
	bytes, err := machinehealthcheckYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "machinehealthcheck.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"machinehealthcheck.yaml": machinehealthcheckYaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"machinehealthcheck.yaml": {machinehealthcheckYaml, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
