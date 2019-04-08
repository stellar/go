package tickerdb

import (
	"fmt"
	"reflect"
)

// getDBFieldTags returns all "db" tags for a given struct, optionally excluding the "id".
func getDBFieldTags(model interface{}, excludeID bool) (fields []string) {
	r := reflect.ValueOf(model)
	for i := 0; i < r.Type().NumField(); i++ {
		dbField := r.Type().Field(i).Tag.Get("db")
		if excludeID && dbField == "id" {
			continue
		}
		fields = append(fields, dbField)
	}
	fields = sanitizeFieldNames(fields)
	return
}

// sanitizeFieldNames adds double quotes to each entry on a slice of field names.
func sanitizeFieldNames(fieldNames []string) (sanitizedFields []string) {
	for _, v := range fieldNames {
		quotedField := fmt.Sprintf("\"%s\"", v)
		sanitizedFields = append(sanitizedFields, quotedField)
	}
	return
}

// getDBFieldValues returns all "db"-tagged values, optionally excluding the "id".
func getDBFieldValues(model interface{}, excludeID bool) (values []interface{}) {
	r := reflect.ValueOf(model)
	for i := 0; i < r.Type().NumField(); i++ {
		dbField := r.Type().Field(i).Tag.Get("db")
		dbVal := r.Field(i).Interface()
		if excludeID && dbField == "id" {
			continue
		}
		values = append(values, dbVal)
	}
	return
}

// createOnConflictFragment generates a ON CONFLICT sql clause for a given constraint,
// preserving the fields listed in the fields param.
func createOnConflictFragment(constraint string, fields []string) (fragment string) {
	fragment = fmt.Sprintf("ON CONFLICT ON CONSTRAINT %s DO UPDATE SET ", constraint)
	for i, field := range fields {
		fragment += fmt.Sprintf("%s = EXCLUDED.%s", field, field)

		if i != len(fields)-1 {
			fragment += ","
		} else {
			fragment += ";"
		}
	}

	return
}

// generatePlaceholders generates a string formatted as (?, ?, ?, ?) of length
// equal to the size of the fields param
func generatePlaceholders(fields []interface{}) (placeholder string) {
	for i := range fields {
		placeholder += "?"

		if i != len(fields)-1 {
			placeholder += ", "
		}
	}
	return
}
