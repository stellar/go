package attachment

import (
	"encoding/json"
)

// Marshal marshals Attachment
func (attachment *Attachment) Marshal() []byte {
	json, _ := json.MarshalIndent(attachment, "", "  ")
	return json
}
