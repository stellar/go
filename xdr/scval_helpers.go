package xdr

import "encoding/base64"

// SerializeScVal converts an ScVal to two map representations:
// 1. A map with base64-encoded value and type
// 2. A map with human-readable decoded value and type
// This is useful for data export and analytics purposes.
func (scVal ScVal) Serialize() (map[string]string, map[string]string) {
	serializedData := map[string]string{}
	serializedData["value"] = "n/a"
	serializedData["type"] = "n/a"

	serializedDataDecoded := map[string]string{}
	serializedDataDecoded["value"] = "n/a"
	serializedDataDecoded["type"] = "n/a"

	if scValTypeName, ok := scVal.ArmForSwitch(int32(scVal.Type)); ok {
		serializedData["type"] = scValTypeName
		serializedDataDecoded["type"] = scValTypeName
		if raw, err := scVal.MarshalBinary(); err == nil {
			serializedData["value"] = base64.StdEncoding.EncodeToString(raw)
			serializedDataDecoded["value"] = scVal.String()
		}
	}

	return serializedData, serializedDataDecoded
}

// SerializeScValArray converts an array of ScVal to two arrays of map representations.
// Each ScVal is serialized using the Serialize method.
func SerializeScValArray(scVals []ScVal) ([]map[string]string, []map[string]string) {
	data := make([]map[string]string, 0, len(scVals))
	dataDecoded := make([]map[string]string, 0, len(scVals))

	for _, scVal := range scVals {
		serializedData, serializedDataDecoded := scVal.Serialize()
		data = append(data, serializedData)
		dataDecoded = append(dataDecoded, serializedDataDecoded)
	}

	return data, dataDecoded
}
