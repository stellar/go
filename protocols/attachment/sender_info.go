package attachment

import (
	"encoding/json"
)

// Marshal marshals SenderInfo
func (senderInfo *SenderInfo) Marshal() []byte {
	json, _ := json.MarshalIndent(senderInfo, "", "  ")
	return json
}
