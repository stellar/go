package protocols

import (
	"net/url"
)

// Asset represents asset
type Asset struct {
	Type   string `name:"asset_type" json:"type"`
	Code   string `name:"asset_code" json:"code"`
	Issuer string `name:"asset_issuer" json:"issuer"`
}

// ForwardDestination contains fields required to create forward federation request
type ForwardDestination struct {
	Domain string     `name:"domain"`
	Fields url.Values `name:"fields"`
}
