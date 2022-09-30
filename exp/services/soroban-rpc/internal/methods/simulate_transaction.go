package methods

import (
	"context"
	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/xdr"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"
)

type SimulateTransactionRequest struct {
	InvokeHostFunctionOp string `json:"invoke_host_function_op"`
}

type SimulateTransactionCost struct {
	CPUInstructions uint64 `json:"cpu_insns,string"`
	MemoryBytes     uint64 `json:"mem_bytes,string"`
}

type SimulateTransactionResponse struct {
	Error     string                  `json:"error,omitempty"`
	Result    string                  `json:"result"`
	Footprint string                  `json:"footprint"`
	Cost      SimulateTransactionCost `json:"cost"`
}

// NewSimulateTransactionHandler returns a json rpc handler to execute preflight requests to stellar core
func NewSimulateTransactionHandler(coreClient *stellarcore.Client) jrpc2.Handler {
	return handler.New(func(ctx context.Context, request SimulateTransactionRequest) SimulateTransactionResponse {
		var xdrOp xdr.InvokeHostFunctionOp
		if err := xdr.SafeUnmarshalBase64(request.InvokeHostFunctionOp, &xdrOp); err != nil {
			return SimulateTransactionResponse{
				Error: "Could unmarshal invoke host function op: " + err.Error(),
			}
		}

		coreResponse, err := coreClient.Preflight(ctx, xdrOp)
		if err != nil {
			return SimulateTransactionResponse{
				Error: "Could not submit request to core: " + err.Error(),
			}
		}

		return SimulateTransactionResponse{
			Error:     coreResponse.Detail,
			Result:    coreResponse.Result,
			Footprint: coreResponse.Footprint,
			Cost: SimulateTransactionCost{
				CPUInstructions: coreResponse.CPUInstructions,
				MemoryBytes:     coreResponse.MemoryBytes,
			},
		}
	})
}
