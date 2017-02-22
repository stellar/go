package attachment

import (
	"encoding/json"
	"reflect"
	"strings"
)

// Marshal marshals SenderInfo
func (senderInfo *SenderInfo) Marshal() ([]byte, error) {
	return json.Marshal(senderInfo)
}

// Map transforms SenderInfo to map[string]string for embedding in
// Transaction/Operation struct.
func (senderInfo SenderInfo) Map() map[string]string {
	m := make(map[string]string)
	st := reflect.TypeOf(senderInfo)
	v := reflect.ValueOf(senderInfo)
	for i := 0; i < st.NumField(); i++ {
		field := st.Field(i)
		if jsonTag, ok := field.Tag.Lookup("json"); ok {
			tagElements := strings.Split(jsonTag, ",")
			m[tagElements[0]] = reflect.Indirect(v).FieldByName(field.Name).String()
		}
	}
	return m
}
