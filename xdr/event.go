package xdr

import (
	"encoding/hex"
	"fmt"
)

func (ce ContractEvent) String() string {
	result := ce.Type.String() + "("
	if ce.ContractId != nil {
		result += hex.EncodeToString(ce.ContractId[:]) + ","
	}
	result += ce.Body.String() + ")"
	return result
}

func (eb ContractEventBody) String() string {
	return fmt.Sprintf("%+v", *eb.V0)
}

func (de DiagnosticEvent) String() string {
	return fmt.Sprintf("%s, successful call: %t", de.Event, de.InSuccessfulContractCall)
}

// GetTopics extracts the topics from a contract event body.
// This is a helper function to abstract the versioning of ContractEventBody.
func (eb ContractEventBody) GetTopics() []ScVal {
	switch eb.V {
	case 0:
		return eb.MustV0().Topics
	default:
		panic("unsupported event body version: " + fmt.Sprintf("%d", eb.V))
	}
}

// GetData extracts the data from a contract event body.
// This is a helper function to abstract the versioning of ContractEventBody.
func (eb ContractEventBody) GetData() ScVal {
	switch eb.V {
	case 0:
		return eb.MustV0().Data
	default:
		panic("unsupported event body version: " + fmt.Sprintf("%d", eb.V))
	}
}
