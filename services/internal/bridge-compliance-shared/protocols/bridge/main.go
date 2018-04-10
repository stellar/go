package bridge

import (
	"regexp"
)

var federationDestinationFieldName = regexp.MustCompile("forward_destination\\[fields\\]\\[([a-z_-]+)\\]")

const (
	pathCodeField   = "path[%d][asset_code]"
	pathIssuerField = "path[%d][asset_issuer]"
)
