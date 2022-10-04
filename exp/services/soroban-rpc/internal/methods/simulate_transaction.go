package methods

import (
	"context"

	"github.com/stellar/go/clients/stellarcore"
	proto "github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"
)

type SimulateTransactionRequest struct {
	InvokeHostFunctionOp string `json:"invoke_host_function_op"`
}

type SimulateTransactionCost struct {
	CPUInstructions uint64 `json:"cpuInsns,string"`
	MemoryBytes     uint64 `json:"memBytes,string"`
}

type SimulateTransactionResponse struct {
	Error     string                  `json:"error,omitempty"`
	Result    string                  `json:"result"`
	Footprint string                  `json:"footprint"`
	Cost      SimulateTransactionCost `json:"cost"`
}

// NewSimulateTransactionHandler returns a json rpc handler to execute preflight requests to stellar core
func NewSimulateTransactionHandler(logger *log.Entry, coreClient *stellarcore.Client) jrpc2.Handler {
	return handler.New(func(ctx context.Context, request SimulateTransactionRequest) SimulateTransactionResponse {
		var xdrOp xdr.InvokeHostFunctionOp
		if err := xdr.SafeUnmarshalBase64(request.InvokeHostFunctionOp, &xdrOp); err != nil {
			logger.WithError(err).WithField("request", request).
				Info("could not unmarshal invoke host function op")
			return SimulateTransactionResponse{
				Error: "Could not unmarshal invoke host function op",
			}
		}

		coreResponse, err := coreClient.Preflight(ctx, xdrOp)
		if err != nil {
			logger.WithError(err).WithField("request", request).
				Info("could not submit preflight request to core")
			return SimulateTransactionResponse{
				Error: "Could not submit request to core",
			}
		}

		if coreResponse.Status == proto.PreflightStatusError {
			return SimulateTransactionResponse{
				Error: coreResponse.Detail,
			}
		}

		return SimulateTransactionResponse{
			Result:    coreResponse.Result,
			Footprint: coreResponse.Footprint,
			Cost: SimulateTransactionCost{
				CPUInstructions: coreResponse.CPUInstructions,
				MemoryBytes:     coreResponse.MemoryBytes,
			},
		}
	})
}
