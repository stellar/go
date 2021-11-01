// Code generated by go-bindata. DO NOT EDIT.
// sources:
// graphiql.html (1.182kB)
// schema.gql (2.42kB)

package static

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
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
		return nil, fmt.Errorf("read %q: %w", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("read %q: %w", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes  []byte
	info   os.FileInfo
	digest [sha256.Size]byte
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
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
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _graphiqlHtml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xb4\x54\x4f\x6f\x13\x3f\x10\x3d\x6f\x3e\x85\x7f\x96\x7e\xd2\x46\x2a\x76\x52\x24\x0e\x9b\x4d\x0e\xd0\xaa\x02\x15\x4a\x81\x0b\x47\xd7\x9e\x5d\x3b\x78\xed\xed\xd8\x9b\x36\xaa\xf2\xdd\x91\xf7\x4f\x28\x7f\x2a\x21\x04\x17\xaf\x3d\x7e\xf3\xde\xf3\xcc\x68\xcb\xff\xce\xae\x5e\x7d\xfa\xfc\xfe\x9c\xe8\xd8\xd8\xcd\xac\x1c\x3e\x59\xa9\x41\xa8\xcd\x2c\xcb\x4a\x6b\xdc\x17\x82\x60\xd7\x34\xc4\xbd\x85\xa0\x01\x22\x25\x1a\xa1\x5a\x53\x1d\x63\x1b\x0a\xce\xa5\x72\xdb\xc0\xa4\xf5\x9d\xaa\xac\x40\x60\xd2\x37\x5c\x6c\xc5\x3d\xb7\xe6\x26\xf0\x1a\x45\xab\xcd\xad\xe5\x0b\xb6\x5c\xb2\xe5\xf2\x18\x60\x32\x04\xca\x7b\x99\x20\xd1\xb4\x91\x04\x94\xbf\x4d\x5b\x41\x94\x9a\x9f\xb2\x05\x7b\x3e\xec\x59\x63\x1c\xdb\x06\xba\x29\xf9\x40\xf7\xa7\xcc\x08\x42\x46\xbe\x7c\xc1\x4e\xd9\x82\x77\x8d\x1a\x02\xac\x45\xaf\x3a\x19\x8d\x77\x7f\x57\xe9\x99\xf2\xcd\x4f\x6a\x29\xf8\x2f\x14\x9f\x6e\xc6\x2f\x14\x4a\x3e\xce\x41\x79\xe3\xd5\x9e\xf4\x13\xb0\xa6\x77\x46\x45\x5d\x90\xe5\x62\xf1\xff\x8a\x68\x30\xb5\x8e\xd3\xa9\x11\x58\x1b\x57\x90\xc5\x8a\xf8\x1d\x60\x65\xfd\x5d\x41\xb4\x51\x0a\xdc\x8a\xf6\x96\x95\xd9\x11\xa3\xd6\x74\x92\xa5\x13\xeb\x23\xa2\x9d\x5e\xd1\xcd\xa5\x17\xca\xb8\x9a\x31\x56\x72\x65\x76\x8f\xde\x9b\xb6\x59\xd5\xb9\xbe\x30\xa4\x6f\xfd\xc5\xf5\x65\xde\x0a\x14\x4d\x98\x93\x87\x74\x9d\x21\xc4\x0e\xc7\xdb\x9c\x0e\xaf\xbc\xb5\xf4\x64\xbc\xce\x1a\x88\xda\xab\x82\xd0\xd6\x87\x48\x4f\x86\x60\x7a\x65\x41\xde\x7c\xbc\x7a\xc7\x42\x44\xe3\x6a\x53\xed\x27\xde\x11\x22\x11\x14\xb8\x68\x84\x0d\x05\xa1\xc6\x49\xdb\x29\x18\xf3\x0f\x73\x16\x35\xb8\xfc\xe8\x2d\x47\x08\xed\xe4\x68\xb2\x94\x62\x2c\xc2\x7d\xcc\xe7\xab\x27\xd2\x92\x8f\x63\x5a\xc4\xfd\xb4\x9d\x28\x7a\x87\xad\xc0\x00\x03\x74\xe0\xc9\x0e\x44\x8a\x28\x35\xc9\x01\xd1\xe3\xfc\xc7\xac\x04\x9d\x90\xa3\x70\x7f\x3c\xcc\xd2\xfa\x21\x4d\xdd\xd9\xd5\x5b\x86\xe0\x14\x60\xde\x23\xfa\x20\x93\x08\x22\xc2\xb9\x85\x06\x5c\xcc\x2f\xfa\xce\x5d\x5f\x9e\x90\x87\xbe\xba\x80\xc5\xb1\x09\x87\xb1\x4c\xca\xcb\x2e\x81\x59\x0d\x71\xcc\x7b\xb9\x7f\xad\xf2\x6f\x6d\x9f\x27\x5c\x5a\xbe\x1b\xb7\x64\x71\x33\x2b\xf9\xf0\x1b\xfa\x1a\x00\x00\xff\xff\xdb\x8e\x2c\x18\x9e\x04\x00\x00")

func graphiqlHtmlBytes() ([]byte, error) {
	return bindataRead(
		_graphiqlHtml,
		"graphiql.html",
	)
}

func graphiqlHtml() (*asset, error) {
	bytes, err := graphiqlHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "graphiql.html", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x76, 0x8, 0xb4, 0x3a, 0xe7, 0xdb, 0xc8, 0x3d, 0x2d, 0x1f, 0x1c, 0x2d, 0xd3, 0x9b, 0xf2, 0xd8, 0xe5, 0xd6, 0x5f, 0x3a, 0x7c, 0x6d, 0x80, 0xf7, 0x40, 0xdc, 0x58, 0xf1, 0x75, 0xbd, 0xf0, 0xa1}}
	return a, nil
}

var _schemaGql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xe4\x54\x51\x6f\xe3\x36\x0c\x7e\xb6\x7f\x05\xdb\xbd\xb4\x2f\x79\x18\xf6\x64\x6c\x03\xd2\x76\xc3\x8a\x35\xb7\xdb\xa5\x37\x0c\x28\x86\x81\xb1\x18\x87\x88\x2c\xf9\x28\x29\x6d\x70\xe8\x7f\x1f\x24\x27\xa9\x6c\xb7\xdd\x0f\xb8\x27\x5b\x24\x3f\x8a\xfc\xf4\x91\xae\xde\x50\x8b\xf0\xb5\x2c\xbe\x04\x92\x7d\x05\xc5\x9f\xf1\x5b\x3e\x97\xa5\xdf\x77\x04\xe9\x14\xdd\xdf\x81\x90\x17\xa6\x1d\x01\x6a\x0d\x3b\xd4\xac\xd0\x93\x02\x74\x8e\xbc\x03\x6b\xc0\x6f\x08\x96\x9e\xb4\x46\x01\x43\xfe\xd1\xca\x76\x56\x16\xbd\xbf\x82\x87\x79\xfc\x39\xfb\xe7\xac\x7c\x27\x19\x3b\x17\x48\xde\xc9\x76\x08\xa8\xe0\xe1\x36\xfd\x4d\xf2\x79\x41\x45\xe0\x3c\x7a\x07\x6b\xb1\x6d\xca\xa3\xd1\x79\xf8\xd1\x84\xf6\x37\x1b\xc4\xcd\x1b\xfb\x33\x6c\xe2\x5f\x44\x5e\x28\x5a\x63\xd0\x1e\x7e\x82\xef\x7f\xe8\xcd\x97\x33\xb0\x9d\x67\x6b\x50\xeb\x3d\x74\x62\x77\xac\x08\x6a\x1b\x8c\x27\x01\x34\x2a\xe2\x56\xe8\xa8\x6f\x1e\xd8\xac\x2d\xac\xad\xc0\x9a\xb5\x27\x61\xd3\xcc\xca\xa2\x45\xd9\x92\x77\x17\x65\x51\xc4\xd0\xd4\xfd\xb5\x55\x54\xc1\xd2\xc7\x90\xdc\xde\xf7\x92\x79\x0e\x77\xbd\x06\xca\x5d\x13\x5c\xd6\x62\x05\xb7\xc6\x97\xc5\x65\x05\x0f\x8b\x54\xca\x84\xf9\xa6\x11\x6a\x12\xed\x03\xd2\xac\xbc\xc1\x59\x44\x27\x7e\x5e\xa5\x07\xa1\x43\x96\x0f\xd8\x12\x5c\xd0\xac\x99\xc1\xf9\xdf\x77\x8b\x7f\xaf\xee\xaf\xcf\xc1\x0a\x20\x44\xb4\x63\xd3\x68\x82\x3a\x88\x90\xa9\xf7\x59\xe0\xf9\xe5\x90\x40\x10\x72\x41\x7b\x37\x2b\x0b\xcf\xf5\x96\x24\xf2\x78\xbc\xe0\x7f\x1b\x9e\x9f\x5a\x3b\xb5\xfe\x5c\x96\xae\xc6\x28\xa6\x2b\x6e\x62\xe0\xe1\x74\xcf\x2d\x1d\xb4\x9e\x28\x8d\x5a\xaf\x33\xc6\xcf\x8e\x9a\x9b\xd7\x89\xf9\xcc\x1e\x41\xd9\xd1\x84\xf6\x10\xe3\x52\x29\x67\x65\x81\xc1\x6f\x3e\xd1\x97\xc0\x42\xaa\x82\x2b\x6b\x35\xa1\x39\xd9\x77\xb6\xc6\x95\xa6\x81\xa3\xed\xef\xf8\x55\x5b\x4c\x09\x7a\x01\x18\x2f\x56\x6b\x52\x57\xfb\x1b\xdb\x22\x9b\x01\xc4\xd4\x1b\x3b\x55\xca\xd0\x73\x3f\x2c\x95\x5d\xb2\xce\x53\xc0\xb0\x34\xc5\xae\xd3\xb8\xbf\xa1\x9a\x5b\xd4\xae\x3a\xd0\x15\xfb\xcb\x98\x8f\x81\xe4\xea\xec\x58\x5b\xa3\x38\x8a\xc2\x65\xc6\x35\x3f\x91\xfa\x10\xda\x55\x14\xe9\x29\x51\x8b\x4f\x13\x1b\xbb\xcf\x46\x73\xcb\x7e\x58\x8d\x90\xa2\x36\x69\xed\xd6\x38\x2f\xa1\x1e\xdf\x50\x5b\xad\xd1\x93\xa0\x9e\x2b\x25\xe4\x1c\xbd\xeb\x5d\x72\x63\xd0\x07\x19\x45\x05\x13\x67\x22\xb7\xc5\x59\x08\x6e\x22\x82\xdb\x9b\xc3\xd3\x1e\xf7\x63\xaf\xaf\x28\x9a\x34\x43\x1f\x91\x25\x03\xbd\x3a\xf8\xb9\x7d\x38\xc0\xc7\x5a\x5e\x19\xfc\x91\x6b\x82\x8b\x19\xff\xb2\x3a\xc4\x27\x3a\x8a\xe7\x00\x18\x9b\x53\xa1\xd7\xbd\xce\x7a\xf2\x6d\x47\xe6\xc5\xaf\xed\xe3\xcb\x61\xc3\xcd\x26\xcb\xb8\x41\xd3\xe4\x37\x68\xeb\xb2\x23\xc7\xeb\x76\xa8\x97\x1e\xc5\x57\x69\xb4\x92\x08\xc4\xf9\x3b\x52\x0d\xc9\x75\x8c\x8f\xe6\x93\x33\x6e\x99\xb7\x7c\x56\x14\xc9\xca\xda\xed\x32\x2e\xa6\x0a\xfe\x18\x9c\x5f\xde\x60\x3c\xed\xef\xbd\xc6\xb7\xca\xd1\xd0\x0e\x5f\x4b\x28\x56\xac\x0e\x1d\x9e\xa6\x70\xc5\x6a\xcc\xc4\x8a\xd5\x02\x9f\xf2\x8d\xb4\x1d\xa3\xd0\x6d\xc7\x28\x74\xdb\x05\x67\x7c\xb9\x4e\x08\xd5\xf8\xbc\x60\xf5\xd1\x72\xb6\xef\x8e\xd5\xf6\xf2\x8e\xef\xd8\x85\x95\xe6\xfa\x77\xda\xe7\x8b\x76\xb8\x88\x82\xe8\x7c\x29\xdb\x56\x7f\xfe\x74\x97\x2f\x21\x52\x24\x18\x17\xc7\x92\x64\x37\x98\x9a\xb8\x87\x27\x46\x2f\x68\xdc\x9a\x64\xe2\x78\xa4\xd5\x3c\xf8\xcd\x2f\x46\x75\x7d\xd5\xd9\x2e\xec\xac\x63\x3f\x41\x58\x69\xee\x1f\xd9\xfb\xdc\xf8\x5c\xfe\x17\x00\x00\xff\xff\x20\x26\x65\x82\x74\x09\x00\x00")

func schemaGqlBytes() ([]byte, error) {
	return bindataRead(
		_schemaGql,
		"schema.gql",
	)
}

func schemaGql() (*asset, error) {
	bytes, err := schemaGqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "schema.gql", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xf7, 0x3c, 0xa2, 0xb0, 0x7, 0x29, 0xef, 0xb6, 0x66, 0x0, 0x23, 0x1e, 0xf8, 0xf4, 0x6a, 0x79, 0x4f, 0xa5, 0xd8, 0xec, 0x24, 0x85, 0xfe, 0xa, 0xfd, 0xe9, 0xc6, 0x27, 0x6c, 0xf1, 0xd8, 0x46}}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetString returns the asset contents as a string (instead of a []byte).
func AssetString(name string) (string, error) {
	data, err := Asset(name)
	return string(data), err
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

// MustAssetString is like AssetString but panics when Asset would return an
// error. It simplifies safe initialization of global variables.
func MustAssetString(name string) string {
	return string(MustAsset(name))
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetDigest returns the digest of the file with the given name. It returns an
// error if the asset could not be found or the digest could not be loaded.
func AssetDigest(name string) ([sha256.Size]byte, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return [sha256.Size]byte{}, fmt.Errorf("AssetDigest %s can't read by error: %v", name, err)
		}
		return a.digest, nil
	}
	return [sha256.Size]byte{}, fmt.Errorf("AssetDigest %s not found", name)
}

// Digests returns a map of all known files and their checksums.
func Digests() (map[string][sha256.Size]byte, error) {
	mp := make(map[string][sha256.Size]byte, len(_bindata))
	for name := range _bindata {
		a, err := _bindata[name]()
		if err != nil {
			return nil, err
		}
		mp[name] = a.digest
	}
	return mp, nil
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
	"graphiql.html": graphiqlHtml,
	"schema.gql":    schemaGql,
}

// AssetDebug is true if the assets were built with the debug flag enabled.
const AssetDebug = false

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"},
// AssetDir("data/img") would return []string{"a.png", "b.png"},
// AssetDir("foo.txt") and AssetDir("notexist") would return an error, and
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		canonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(canonicalName, "/")
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
	"graphiql.html": {graphiqlHtml, map[string]*bintree{}},
	"schema.gql":    {schemaGql, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory.
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

// RestoreAssets restores an asset under the given directory recursively.
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
	canonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(canonicalName, "/")...)...)
}
