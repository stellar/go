package horizon

import (
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/support/render/hal"
)

// DataShowAction renders a account summary found by its address.
type DataShowAction struct {
	Action
	Address string
	Key     string
	Data    core.AccountData
}

// JSON is a method for actions.JSON
func (action *DataShowAction) JSON() {
	action.Do(
		action.loadParams,
		action.loadRecord,
		func() {

			hal.Render(action.W, map[string]string{
				"value": action.Data.Value,
			})
		},
	)
}

// Raw is a method for actions.Raw
func (action *DataShowAction) Raw() {
	action.Do(
		action.loadParams,
		action.loadRecord,
		func() {
			var raw []byte
			raw, action.Err = action.Data.Raw()
			if action.Err != nil {
				return
			}

			action.W.Write(raw)
		},
	)
}

// SetupAndValidateSSE calls the setup functions before we can stream and validates
// the request parameters. Errors are stored in action.Err.
func (action *DataShowAction) SetupAndValidateSSE() {
	action.Setup(
		action.loadParams,
		action.loadRecord,
	)
}

// SSE is a method for actions.SSE that loads the latest account data and sends them to stream.
func (action *DataShowAction) SSE(stream sse.Stream) {
	functionsToExecute := []func(){nil}
	// No point reloading data if Setup was just called.
	if action.InitialDataIsFresh == false {
		functionsToExecute = append(functionsToExecute, action.loadParams, action.loadRecord)
	} else {
		action.InitialDataIsFresh = false
	}
	functionsToExecute = append(functionsToExecute, func() {
		stream.Send(sse.Event{Data: action.Data.Value})
	})
	action.Do(functionsToExecute...)
}

func (action *DataShowAction) loadParams() {
	action.Address = action.GetString("account_id")
	action.Key = action.GetString("key")
}

func (action *DataShowAction) loadRecord() {
	action.Err = action.CoreQ().
		AccountDataByKey(&action.Data, action.Address, action.Key)
	if action.Err != nil {
		return
	}
}
