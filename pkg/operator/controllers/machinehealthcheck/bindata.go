// Code generated for package machinehealthcheck by go-bindata DO NOT EDIT. (@generated)
// sources:
// staticresources/machinehealthcheck.yaml
// staticresources/mhcremediationalert.yaml
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

var _machinehealthcheckYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x9c\x90\xb1\x6e\xf3\x30\x0c\x84\x77\x3f\x05\xa1\xdd\xf9\x13\xfc\x9b\xd6\x20\x45\x3b\xb4\x43\x81\x74\x67\x65\x06\x16\x2c\x51\x82\x48\xa7\xf1\xdb\x17\x8a\xe2\x74\x28\xba\x64\x93\x4e\xc7\xef\x4e\xc4\xec\x3f\xa8\x88\x4f\x6c\x21\xa2\x1b\x3d\xd3\x26\x65\x62\x19\xfd\x49\x37\x3e\xfd\x3b\xef\x3e\x49\x71\xd7\x4d\x9e\x07\x0b\xaf\xcd\xf2\x4c\x18\x74\xdc\x8f\xe4\xa6\x2e\x92\xe2\x80\x8a\xb6\x03\x60\x8c\x64\x01\x4b\xea\x6f\xac\xf1\x6a\x74\x57\x63\x7b\x96\x8c\x8e\x2c\xdc\x33\x56\x67\x8f\xd9\x77\x92\xc9\x55\x8e\x50\x20\xa7\xa9\xd4\x33\x40\x44\x75\xe3\xe1\x92\x0b\x49\x2d\x2a\x4d\xed\x61\xa2\xe5\x8f\xd2\x2e\xcc\xa2\x54\x2a\xf3\xce\x2f\x29\xd0\x75\x10\x6a\x78\xc1\x8a\x87\xb7\xa4\x2f\x7c\x53\xcf\x18\x66\xba\xc1\x2b\xde\x78\x3e\x15\x34\x3f\xf7\x88\x15\x6a\x1e\x49\x17\xd2\x5f\xd9\x87\x8b\x17\x95\x0e\x60\xe6\xb6\xa6\x65\x9f\x78\xf0\xba\x7e\xb1\x07\x5d\x32\x59\x30\xef\x84\xc3\xd2\x62\xd5\x47\x4a\xb3\x5a\x30\xff\xb7\x5b\x69\x9a\x28\xea\x2c\x16\xcc\x13\x06\x21\xf3\xc8\xe4\x91\x27\x4e\x5f\x5c\xd5\x88\x97\xe3\xda\xc7\x82\xd9\x99\xee\x3b\x00\x00\xff\xff\x38\x97\xb5\xaf\x23\x02\x00\x00")

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

var _mhcremediationalertYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x74\x91\x4d\x8b\x14\x4d\x10\x84\xef\xfd\x2b\xe2\xf8\xbe\x87\x5e\x5d\x0f\x22\x75\x10\x44\x84\xbd\x2c\xc8\x08\x5e\x44\x86\x9c\xea\x70\xaa\x98\xfa\x68\x32\xb3\x67\xdd\x7f\x2f\xdd\xd3\xea\x5e\xcc\x63\x55\x3e\x11\x4f\x51\x32\xe7\xaf\x54\xcb\xbd\x05\xd4\xde\xb2\x77\xcd\xed\x7c\x17\xbb\xb2\xdb\x5d\xec\xf5\xd5\xf5\x7e\xb8\xe4\x36\x05\x7c\xd6\x5e\xe9\x89\x8b\x1d\x96\xc2\xa1\xd2\x65\x12\x97\x30\x00\x4d\x2a\x03\x6a\x8a\xa3\xb2\x72\xca\xe2\xb9\xb7\x51\x0a\xd5\xf7\x5b\x9b\x25\x32\xa0\xcf\x6c\x96\xf2\x0f\x1f\xab\xc4\x94\x1b\x47\x99\xf3\x00\x14\x39\xb1\xd8\x1a\x05\xcc\x7f\x7a\x02\x2e\xef\x6c\x3b\xd3\x5e\x18\xb0\x05\x8e\xba\x14\xda\x60\x33\xe3\xba\x7f\xd6\xbe\xcc\x1b\x39\xee\x1a\xa6\x1c\xff\xa5\x02\x6c\xf4\xad\x68\xbc\x05\x06\x7c\x39\x7c\x7a\xbc\xe9\x3c\x50\x8a\xa7\x8f\x89\xf1\x72\xf8\x8b\x1f\xc4\xf9\x90\xcf\x69\xa3\x00\xfe\x9c\x35\x20\xb7\xa8\x14\xe3\x7f\x55\xe6\x7c\xdc\x9f\x93\x36\x3e\xae\xfc\xf1\x45\xff\xd1\x96\x18\x69\x76\xf4\xee\x52\xf0\xed\xed\xeb\xfa\xfd\x7f\xbc\xc7\xfd\x9e\xf8\xa1\xb5\xee\xdb\xe6\xae\xb6\xce\x23\xcd\xe4\xcc\x80\xa7\xae\x17\x2a\x5a\x9f\x68\x48\x72\x25\x4e\x64\xc3\xef\x7c\x4e\x78\x83\xae\xa8\x5d\x09\xcf\x95\x86\xdc\xe0\x89\x28\x62\x8e\xd4\x17\x85\xa7\x6c\xa8\xf2\x8c\xdc\xa6\x1c\xc5\x09\x69\x58\x9a\xb9\x9c\x0a\xb7\x82\xd2\x65\x82\x2e\xad\xe5\x76\x46\xbf\xf1\xb1\x2c\xe6\xd4\xdd\xe8\xe5\x1f\xad\x63\xbc\x52\xb3\x3f\x07\x3c\x89\xae\xd8\xf0\x2b\x00\x00\xff\xff\x9e\xca\xa1\xb4\x4d\x02\x00\x00")

func mhcremediationalertYamlBytes() ([]byte, error) {
	return bindataRead(
		_mhcremediationalertYaml,
		"mhcremediationalert.yaml",
	)
}

func mhcremediationalertYaml() (*asset, error) {
	bytes, err := mhcremediationalertYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "mhcremediationalert.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
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
	"machinehealthcheck.yaml":  machinehealthcheckYaml,
	"mhcremediationalert.yaml": mhcremediationalertYaml,
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
	"machinehealthcheck.yaml":  {machinehealthcheckYaml, map[string]*bintree{}},
	"mhcremediationalert.yaml": {mhcremediationalertYaml, map[string]*bintree{}},
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
