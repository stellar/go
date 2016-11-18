package db

import (
	"reflect"

	"github.com/jmoiron/sqlx/reflectx"
)

var mapper = reflectx.NewMapper("db")

type person struct {
	Name        string `db:"name"`
	HungerLevel string `db:"hunger_level"`

	SomethingIgnored int `db:"-"`
}

// columnsForStruct returns a slice of column names for the provided value
// (which should be a struct, a slice of structs).
func columnsForStruct(dest interface{}) []string {
	typ := reflect.TypeOf(dest)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}

	typmap := mapper.TypeMap(typ)

	var keys []string
	for k := range typmap.Names {
		keys = append(keys, k)
	}
	return keys
}
