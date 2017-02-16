package attachment

import (
	"encoding/json"
)

// Marshal marshals SenderInfo
func (senderInfo *SenderInfo) Marshal() ([]byte, error) {
	return json.Marshal(senderInfo)
}
