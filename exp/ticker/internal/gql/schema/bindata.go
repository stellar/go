// Code generated by go-bindata. DO NOT EDIT.
// sources:
// schema.gql

package schema


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
	info  fileInfoEx
}

type fileInfoEx interface {
	os.FileInfo
	MD5Checksum() string
}

type bindataFileInfo struct {
	name        string
	size        int64
	mode        os.FileMode
	modTime     time.Time
	md5checksum string
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) MD5Checksum() string {
	return fi.md5checksum
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _bindataSchemagql = []byte(
	"\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xd4\x54\x4f\x6f\xfb\x36\x0c\x3d\xdb\x9f\x82\xc5\xef\xd2\x5e\x72\x18\x76" +
	"\x32\xb6\x01\x49\xbb\x61\xc1\x7e\x2d\xb6\x26\x1d\x06\x14\xc3\xc0\x58\xb4\x2d\x54\x7f\x5c\x4a\x4a\x1a\x0c\xfd\xee" +
	"\x83\xe4\x24\x95\xe3\x2e\x3b\xef\x64\x89\xe4\xa3\xf4\x9e\x1f\xe5\xea\x8e\x34\xc2\xdf\x65\xf1\x1a\x88\xf7\x15\x14" +
	"\xbf\xc5\x6f\xf9\x5e\x96\x7e\xdf\x13\xa4\x5d\x4c\x7f\x01\x26\xcf\x92\xb6\x04\xa8\x14\x6c\x51\x49\x81\x9e\x04\xa0" +
	"\x73\xe4\x1d\x58\x03\xbe\x23\x58\x79\x52\x0a\x19\x0c\xf9\x9d\xe5\x97\x59\x59\x0c\xf9\x0a\x9e\xe7\x71\xf1\xe7\x55" +
	"\x79\xa1\x97\x74\x2e\x10\x5f\x68\x76\x28\xa8\xe0\x79\x99\x56\xe7\xed\x3c\xa3\x20\x70\x1e\xbd\x83\x86\xad\x4e\x6d" +
	"\x14\x3a\x0f\xdf\x99\xa0\x7f\xb6\x81\xdd\xbc\xb5\x3f\x40\x17\x57\x11\x79\x2d\xa8\xc1\xa0\x3c\x7c\x0f\xdf\x7c\x3b" +
	"\x84\x6f\x66\x60\x7b\x2f\xad\x41\xa5\xf6\xd0\xb3\xdd\x4a\x41\x50\xdb\x60\x3c\x31\xa0\x11\x11\xb7\x41\x47\x03\x75" +
	"\x90\xa6\xb1\xd0\x58\x86\x46\x2a\x4f\x2c\x4d\x3b\x2b\x0b\x8d\xfc\x42\xde\x5d\x97\x45\x11\x4b\x13\xf7\x5b\x2b\xa8" +
	"\x82\x95\x8f\x25\x79\x7c\xa0\x92\x65\x0e\x67\x7d\x06\xca\x53\x13\x5c\x46\xb1\x82\xa5\xf1\x65\x71\x53\xc1\xf3\x7d" +
	"\xba\xca\x44\xf8\xb6\x65\x6a\x93\xea\x23\xd1\x2c\xff\x8b\x66\x11\x9d\xf4\xf9\x54\x9e\x88\x31\xa8\x09\x6c\x93\xd6" +
	"\x43\xcf\x1e\x25\xc3\x35\xcd\xa2\x22\x5f\xe0\x8f\xaf\xf7\x7f\x2d\xd6\xb7\x37\x63\xb1\x80\xc9\x05\xe5\xdd\xac\x2c" +
	"\xbc\xac\x5f\x88\xa3\x66\x11\xf8\x80\x9a\xfe\x93\xdc\xfc\x44\xe3\x44\xf3\xbd\x2c\x5d\x8d\xd1\x37\x6b\xa9\xe9\xb8" +
	"\x5e\xc8\x36\x82\x06\x57\x27\xf9\xa2\xab\xeb\x4c\xdd\xab\xa3\xbd\xe6\x75\x52\x39\x8b\x47\x50\xb6\x35\x41\x1f\x6a" +
	"\x5c\xba\xca\x55\x59\x60\xf0\xdd\x23\xbd\x06\xc9\x24\x2a\x58\x58\xab\x08\xcd\x29\xbe\xb5\x35\x6e\x14\x8d\x12\x7a" +
	"\x38\xe3\x27\x65\x31\x35\x18\x7e\xb6\xf1\x6c\x95\x22\xb1\xd8\xdf\x59\x8d\xd2\x8c\x20\xa6\xee\xec\xd4\x15\xe3\xcc" +
	"\x7a\x7c\x55\xe9\x52\x74\x9e\x0a\xc6\x57\x13\xd2\xf5\x0a\xf7\x77\x54\x4b\x8d\xca\x55\x07\x89\x22\xbf\x4c\xf9\x58" +
	"\x48\xae\xce\xb6\xb5\x35\x42\x46\x03\xb8\x2c\xd8\xc8\x37\x12\x0f\x41\x6f\xa2\x21\x4f\x8d\x34\xbe\x4d\x62\xd2\x3d" +
	"\x19\x25\xb5\xf4\xe3\xdb\x30\x09\xd2\xc9\x57\x4b\xe3\x3c\x87\xfa\xfc\x84\xda\x2a\x85\x9e\x18\xd5\x5c\x08\x26\xe7" +
	"\xe8\x62\x76\x25\x5b\x83\x3e\xf0\x59\x55\x30\xd1\xff\x79\x2c\xfa\x3e\xb8\x89\x09\x96\x77\x87\x5f\x7b\x7c\x09\x07" +
	"\x7f\x45\xd3\x24\x6f\xff\x8a\x92\x33\xd0\xc7\x30\x1f\x71\xe3\x51\x3d\x45\x63\xe1\xef\x56\x85\xa8\xf0\xf1\xdf\x1f" +
	"\x2a\xcf\xc3\xe9\x9c\xdb\xc1\x26\x03\xd8\xf6\x64\x3e\xf2\xca\xee\x3e\x36\x9d\x6c\xbb\xac\x63\x87\xa6\xcd\x4f\x50" +
	"\xd6\x9d\x6f\xe3\x74\x54\x69\x46\xa2\x0a\xd2\xd4\xa7\xdd\x91\xf2\xf9\x70\x5d\x22\xff\x7f\xe1\x34\x3c\x9a\x91\x49" +
	"\x1f\x36\x4a\xd6\xbf\xd0\x3e\x9f\xec\xb1\xf3\x03\xab\xfc\x15\xb0\x5a\x3d\x3d\x7e\xcd\x5d\x4f\x82\x18\xa3\x53\x57" +
	"\xc4\x5b\xca\x35\x89\x83\x3f\x09\x7a\x46\xe3\x1a\xe2\x49\x62\x47\x9b\x79\xf0\xdd\x8f\x46\xf4\x56\x8e\x9e\x1e\x41" +
	"\xbd\x75\xd2\x4f\x10\x96\xdb\xf5\x4e\x7a\x9f\x07\xdf\xcb\x7f\x02\x00\x00\xff\xff\xcf\x20\x3f\xff\xcf\x07\x00\x00" +
	"")

func bindataSchemagqlBytes() ([]byte, error) {
	return bindataRead(
		_bindataSchemagql,
		"schema.gql",
	)
}



func bindataSchemagql() (*asset, error) {
	bytes, err := bindataSchemagqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{
		name: "schema.gql",
		size: 1999,
		md5checksum: "",
		mode: os.FileMode(420),
		modTime: time.Unix(1555715632, 0),
	}

	a := &asset{bytes: bytes, info: info}

	return a, nil
}


//
// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
//
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

//
// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
// nolint: deadcode
//
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

//
// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or could not be loaded.
//
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

//
// AssetNames returns the names of the assets.
// nolint: deadcode
//
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

//
// _bindata is a table, holding each asset generator, mapped to its name.
//
var _bindata = map[string]func() (*asset, error){
	"schema.gql": bindataSchemagql,
}

//
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
//
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, &os.PathError{
					Op: "open",
					Path: name,
					Err: os.ErrNotExist,
				}
			}
		}
	}
	if node.Func != nil {
		return nil, &os.PathError{
			Op: "open",
			Path: name,
			Err: os.ErrNotExist,
		}
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

var _bintree = &bintree{Func: nil, Children: map[string]*bintree{
	"schema.gql": {Func: bindataSchemagql, Children: map[string]*bintree{}},
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
	return os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
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
