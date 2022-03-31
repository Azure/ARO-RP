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

var _machinehealthcheckYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x9c\x90\xb1\x6e\xf3\x30\x0c\x84\x77\x3f\x05\xa1\xdd\xf9\x13\xfc\x9b\xd6\x20\x45\x3b\xb4\x43\x81\x74\x67\x65\x06\x12\x2c\x51\x82\x48\xa7\xf1\xdb\x17\xb2\x93\x74\x28\xba\x64\xb3\x4f\x77\xdf\x9d\x84\x25\x7c\x50\x95\x90\xd9\x42\x42\xe7\x03\xd3\x26\x17\x62\xf1\xe1\xa4\x9b\x90\xff\x9d\x77\x9f\xa4\xb8\xeb\xc6\xc0\x83\x85\xd7\xd5\xf2\x4c\x18\xd5\xef\x3d\xb9\xb1\x4b\xa4\x38\xa0\xa2\xed\x00\x18\x13\x59\xc0\x9a\xfb\x2b\xcb\x2f\x46\xb7\x18\xd7\x63\x29\xe8\xc8\xc2\xbd\xe3\xe6\xec\xb1\x84\x4e\x0a\xb9\xc6\x11\x8a\xe4\x34\xd7\xf6\x0d\x90\x50\x9d\x3f\x5c\x4a\x25\x69\x43\x65\x55\x7b\x18\x69\xfe\x63\xb4\x8b\x93\x28\xd5\xc6\xbc\xf3\x6b\x8e\xb4\x04\xa1\x95\x57\x6c\x78\x78\xcb\xfa\xc2\x57\xf5\x8c\x71\xa2\x2b\xbc\xe1\x4d\xe0\x53\x45\xf3\xf3\x9f\xb0\x41\xcd\x23\xed\x42\xfa\xab\xfb\x70\x09\xa2\xd2\x01\x4c\xbc\x3e\xd3\xbc\xcf\x3c\x04\xbd\x5d\xb1\x07\x9d\x0b\xd9\x16\x32\xef\x84\xc3\xbc\x36\x6b\x48\x94\x27\xb5\x60\xfe\x6f\xb7\x62\x60\x11\x45\x51\x27\xb1\x60\x9e\x30\x0a\x99\x47\xd3\x47\x1e\x39\x7f\x71\xb3\x26\xbc\x1c\x6f\xbb\x2c\x98\x9d\xf9\x0e\x00\x00\xff\xff\x9f\x0e\x45\xac\x2a\x02\x00\x00")

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
