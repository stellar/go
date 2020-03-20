// Code generated by go-bindata. DO NOT EDIT.
// sources:
// migrations/20200309000000-initial-1.sql (162B)
// migrations/20200309000001-initial-2.sql (162B)
// migrations/20200311000000-create-accounts.sql (335B)
// migrations/20200311000001-create-identities.sql (444B)
// migrations/20200311000002-create-auth-methods.sql (961B)

package dbmigrate

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

var _migrations20200309000000Initial1Sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x54\xcc\xd1\x0d\xc2\x30\x0c\x04\xd0\xff\x4c\x71\xff\x28\x4c\xc1\x08\x30\x80\x01\xa7\xb5\xd4\xda\x91\x6d\xa8\xb2\x3d\x8a\xf8\x40\x7c\xde\xdd\xd3\xd5\x8a\xeb\x2a\x81\x5d\x16\xa7\x14\x53\x34\xd9\x18\x12\x10\x4d\xd6\xd9\xd0\xb6\x0d\xf0\xde\x73\x80\xf4\x39\x27\x42\x13\x8f\x44\x24\x79\x8a\x2e\xe8\x26\x9a\x68\xe6\xa5\x56\xd8\xcb\x7f\x77\x81\x3b\x37\x73\xc6\xc1\x18\x9c\x58\xe9\xcd\x20\xc4\x63\xe5\x9d\xce\x65\xfa\xd3\x17\x33\x6e\xfd\x3f\x5f\xec\xd0\x52\x3e\x01\x00\x00\xff\xff\xd3\x79\x21\xda\xa2\x00\x00\x00")

func migrations20200309000000Initial1SqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations20200309000000Initial1Sql,
		"migrations/20200309000000-initial-1.sql",
	)
}

func migrations20200309000000Initial1Sql() (*asset, error) {
	bytes, err := migrations20200309000000Initial1SqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/20200309000000-initial-1.sql", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xd1, 0xd1, 0x21, 0xe9, 0x6d, 0xe0, 0xfe, 0xb4, 0x8b, 0x78, 0x2, 0xae, 0x5c, 0xd5, 0x8b, 0x41, 0xb8, 0x4b, 0xaa, 0x3a, 0xea, 0x69, 0xf, 0xf3, 0x2f, 0x6c, 0xae, 0x38, 0x46, 0xb, 0x2, 0xfc}}
	return a, nil
}

var _migrations20200309000001Initial2Sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x54\xcc\xd1\x0d\xc2\x30\x0c\x04\xd0\xff\x4c\x71\xff\x28\x4c\xc1\x08\x30\x80\x01\xa7\xb5\xd4\xda\x91\x6d\xa8\xb2\x3d\x8a\xf8\x40\x7c\xde\xdd\xd3\xd5\x8a\xeb\x2a\x81\x5d\x16\xa7\x14\x53\x34\xd9\x18\x12\x10\x4d\xd6\xd9\xd0\xb6\x0d\xf0\xde\x73\x80\xf4\x39\x27\x42\x13\x8f\x44\x24\x79\x8a\x2e\xe8\x26\x9a\x68\xe6\xa5\x56\xd8\xcb\x7f\x77\x81\x3b\x37\x73\xc6\xc1\x18\x9c\x58\xe9\xcd\x20\xc4\x63\xe5\x9d\xce\x65\xfa\xd3\x17\x33\x6e\xfd\x3f\x5f\xec\xd0\x52\x3e\x01\x00\x00\xff\xff\xd3\x79\x21\xda\xa2\x00\x00\x00")

func migrations20200309000001Initial2SqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations20200309000001Initial2Sql,
		"migrations/20200309000001-initial-2.sql",
	)
}

func migrations20200309000001Initial2Sql() (*asset, error) {
	bytes, err := migrations20200309000001Initial2SqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/20200309000001-initial-2.sql", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xd1, 0xd1, 0x21, 0xe9, 0x6d, 0xe0, 0xfe, 0xb4, 0x8b, 0x78, 0x2, 0xae, 0x5c, 0xd5, 0x8b, 0x41, 0xb8, 0x4b, 0xaa, 0x3a, 0xea, 0x69, 0xf, 0xf3, 0x2f, 0x6c, 0xae, 0x38, 0x46, 0xb, 0x2, 0xfc}}
	return a, nil
}

var _migrations20200311000000CreateAccountsSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x84\x90\x4f\x4b\xc4\x30\x14\xc4\xef\xef\x53\xcc\xb1\x45\xf7\x13\xf4\x94\xb5\x0f\x37\xd8\xa6\x31\x7d\x61\x77\xbd\x2c\xa1\x09\x52\xd0\x5d\x69\x2b\x7e\x7d\xa1\x68\x09\x5e\x3c\xbe\x3f\xf3\x63\x66\x76\x3b\xdc\xbd\x8f\xaf\x53\x58\x12\xfc\x07\xd1\x83\x63\x25\x0c\x51\xfb\x86\x11\x86\xe1\xf6\x79\x5d\x66\x14\x04\x8c\x11\x7b\xfd\xd8\xb3\xd3\xaa\x81\xe9\x04\xc6\x37\xcd\x3d\x11\x60\x9d\x6e\x95\x3b\xe3\x89\xcf\x28\xc6\x58\xae\xcb\x61\x4a\x61\x49\xf1\x12\x16\x88\x6e\xb9\x17\xd5\x5a\x1c\xb5\x1c\xd6\x11\x2f\x9d\xe1\x8c\x02\xc4\xf4\x96\xfe\xf9\x5f\xb9\x21\xc6\x29\xcd\x33\x84\x4f\xb2\x01\xa8\xac\x36\xeb\xde\xe8\x67\xcf\xd0\xa6\xe6\xd3\x96\xe0\xf2\x2b\xeb\x4c\x96\xca\x5b\xcb\xae\xf8\x39\x95\x25\x8e\x07\x76\x9c\x3b\xd1\xfd\x8a\xaf\x88\xf2\x9e\xea\xdb\xd7\x95\xa8\x76\x9d\xfd\xd3\x53\x45\xdf\x01\x00\x00\xff\xff\x6e\xdf\x97\x68\x4f\x01\x00\x00")

func migrations20200311000000CreateAccountsSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations20200311000000CreateAccountsSql,
		"migrations/20200311000000-create-accounts.sql",
	)
}

func migrations20200311000000CreateAccountsSql() (*asset, error) {
	bytes, err := migrations20200311000000CreateAccountsSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/20200311000000-create-accounts.sql", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x8b, 0x6e, 0x3e, 0x48, 0x39, 0xd6, 0x5, 0xcb, 0xd0, 0x73, 0x7, 0xdc, 0xb8, 0x76, 0x93, 0x8f, 0x2a, 0xde, 0x35, 0xbd, 0x6c, 0xb, 0x1a, 0x1a, 0x3e, 0x3d, 0xf6, 0x79, 0xae, 0x2d, 0xa0, 0xcb}}
	return a, nil
}

var _migrations20200311000001CreateIdentitiesSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x84\x91\xcb\x6e\xab\x30\x10\x86\xf7\xf3\x14\xff\x32\xe8\x24\x4f\xc0\xca\x81\x49\x62\x1d\x30\xc8\x38\x4a\xd2\x0d\x42\xd8\xaa\x2c\xa5\x50\x11\x57\x7d\xfd\x0a\x7a\x89\xdb\x2e\xba\xb4\x67\xf4\xcd\x7f\xd9\x6c\xf0\xef\xc9\x3f\x4e\x5d\x70\x38\x3e\x13\x65\x9a\x85\x61\x18\xb1\x2d\x18\xde\xba\x21\xf8\xe0\xdd\x0d\x2b\x02\xba\xbe\x1f\x5f\x86\xd0\x7a\x8b\xad\xdc\x4b\x65\xa0\x2a\x03\x75\x2c\x8a\x35\x01\xef\xbf\x0d\x6b\x29\x8a\x68\x40\xc0\xae\xd2\x2c\xf7\x0a\xff\xf9\x82\xd5\x1d\x92\x40\xf3\x8e\x35\xab\x8c\x9b\x4f\xf6\x0d\xab\x79\x50\x29\xe4\x5c\xb0\x61\x64\xa2\xc9\x44\xce\xf3\x81\x5a\xcb\x52\xe8\xcb\x4f\xcc\x1a\xde\x26\xcb\x9d\x7e\x72\x5d\x70\xb6\xed\x02\x8c\x2c\xb9\x31\xa2\xac\x71\x92\xe6\xb0\x3c\xf1\x50\x29\xfe\xa6\xd8\xba\xab\xfb\x63\x7f\xe1\x4e\xe3\xd5\xc1\xf0\xf9\xee\x97\x92\xf4\x2b\x2a\xa9\x72\x3e\x47\x51\xb5\x1f\xd2\xbc\x9d\x6d\xc4\x11\xc6\xd6\x4f\x07\xd6\x1c\x2b\x90\xcd\x42\x4e\x89\xe2\x4a\xf2\xf1\x75\x20\xca\x75\x55\xff\xaa\x24\xa5\xb7\x00\x00\x00\xff\xff\x65\x2a\xe7\xa1\xbc\x01\x00\x00")

func migrations20200311000001CreateIdentitiesSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations20200311000001CreateIdentitiesSql,
		"migrations/20200311000001-create-identities.sql",
	)
}

func migrations20200311000001CreateIdentitiesSql() (*asset, error) {
	bytes, err := migrations20200311000001CreateIdentitiesSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/20200311000001-create-identities.sql", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x76, 0x76, 0xdc, 0x95, 0x43, 0x7e, 0x2, 0x98, 0x5, 0xc6, 0xb5, 0x40, 0xc1, 0x36, 0x95, 0x8d, 0x1, 0xb6, 0x28, 0xfa, 0x6e, 0x45, 0xa, 0xb3, 0x0, 0xfb, 0x66, 0xf3, 0xab, 0xf9, 0x45, 0x5a}}
	return a, nil
}

var _migrations20200311000002CreateAuthMethodsSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x9c\x93\x5f\x6f\x9b\x30\x14\xc5\xdf\xfd\x29\xae\xfa\x92\x54\x4b\x3f\x01\x4f\x2e\xdc\xb6\xd6\xc0\x20\xe3\xa8\xcb\x5e\x2c\x2f\xbe\x5a\x90\xf8\x13\x81\xd9\xd6\x6f\x3f\x01\xe9\x0a\x6d\x92\x4d\x7b\xe4\x5c\xf3\x3b\xf6\x39\xba\x77\x77\xf0\xa9\x2a\xbe\xb7\xd6\x13\x6c\x8f\x8c\x85\x0a\xb9\x46\xd0\xbb\x0c\xc1\xf6\xfe\x60\x2a\xf2\x87\xc6\x19\xff\x72\x24\xe0\x39\xa0\xdc\x26\xb0\x66\x00\xab\xce\x53\x59\xda\xd6\x58\xe7\x5a\xea\xba\xd5\x66\x10\x8f\x87\xa6\x26\x53\xf7\xd5\x37\x6a\x27\x85\x2a\x5b\x94\x2b\x76\x1b\xbc\xb1\xf9\x7d\xbc\x80\x77\x23\xd0\xee\xf7\x4d\x5f\x7b\x53\x38\xb8\x17\x8f\x42\x6a\x90\xa9\x06\xb9\x8d\xe3\x81\x53\x38\xaa\x7d\xe1\x5f\x2e\x8e\x07\x35\x47\x25\x78\x3c\x1b\x30\x80\x87\x54\xa1\x78\x94\xf0\x19\x77\xb0\x7e\xf3\xb8\x05\x85\x0f\xa8\x50\x86\x98\xbf\x5a\x77\xb0\x1e\x06\xa9\x84\x08\x63\xd4\x08\x21\xcf\x43\x1e\xe1\xe6\x32\x66\x33\xbf\xd8\x82\x79\xd2\x0b\xea\xde\x9f\xbf\xe0\x90\x29\x91\x70\xb5\xbb\xea\x30\xfe\x3e\x3e\x6b\xdf\x92\xf5\xe4\x8c\xf5\xa0\x45\x82\xb9\xe6\x49\x06\xcf\x42\x3f\x8d\x9f\xf0\x35\x95\xb8\x08\xc8\x51\x49\x7f\x39\x3f\x72\x6f\x86\xa2\x6f\x3e\x56\x3f\x67\xfd\xb0\x65\x4f\xe0\xe9\x97\xff\x23\xcf\xfb\x15\x32\xc2\x2f\x8b\x7e\xcd\xe9\x39\x85\x1b\x9e\xbe\x6c\x7e\x5e\xc9\xf3\x13\x2a\x9c\x5f\x55\xe4\x23\x3d\xf8\x17\xb4\x79\xcd\xe9\xba\xcb\xbb\xc6\xfe\xc7\x72\x08\xc4\x4c\x21\x7c\x30\x9a\xf2\xdb\x4c\x19\x5d\xc3\xb3\xf9\xe6\x45\xcd\xcf\x9a\xb1\x48\xa5\xd9\x99\xed\x08\x4e\x83\x73\x2b\x19\xb0\xdf\x01\x00\x00\xff\xff\xdc\x7b\x89\xb1\xc1\x03\x00\x00")

func migrations20200311000002CreateAuthMethodsSqlBytes() ([]byte, error) {
	return bindataRead(
		_migrations20200311000002CreateAuthMethodsSql,
		"migrations/20200311000002-create-auth-methods.sql",
	)
}

func migrations20200311000002CreateAuthMethodsSql() (*asset, error) {
	bytes, err := migrations20200311000002CreateAuthMethodsSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "migrations/20200311000002-create-auth-methods.sql", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xed, 0xcd, 0x63, 0xde, 0x88, 0xf2, 0x4b, 0xb2, 0x32, 0xe1, 0xd5, 0x42, 0x8b, 0x6e, 0xf7, 0xd4, 0x70, 0xab, 0xba, 0x2b, 0xfc, 0x94, 0x4e, 0x98, 0x6b, 0x39, 0x4e, 0xfd, 0x77, 0xd0, 0x7, 0xb}}
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
	"migrations/20200309000000-initial-1.sql":           migrations20200309000000Initial1Sql,
	"migrations/20200309000001-initial-2.sql":           migrations20200309000001Initial2Sql,
	"migrations/20200311000000-create-accounts.sql":     migrations20200311000000CreateAccountsSql,
	"migrations/20200311000001-create-identities.sql":   migrations20200311000001CreateIdentitiesSql,
	"migrations/20200311000002-create-auth-methods.sql": migrations20200311000002CreateAuthMethodsSql,
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
	"migrations": &bintree{nil, map[string]*bintree{
		"20200309000000-initial-1.sql":           &bintree{migrations20200309000000Initial1Sql, map[string]*bintree{}},
		"20200309000001-initial-2.sql":           &bintree{migrations20200309000001Initial2Sql, map[string]*bintree{}},
		"20200311000000-create-accounts.sql":     &bintree{migrations20200311000000CreateAccountsSql, map[string]*bintree{}},
		"20200311000001-create-identities.sql":   &bintree{migrations20200311000001CreateIdentitiesSql, map[string]*bintree{}},
		"20200311000002-create-auth-methods.sql": &bintree{migrations20200311000002CreateAuthMethodsSql, map[string]*bintree{}},
	}},
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
