package server

import (
	"cli_tool/gen/event_service"
	"log"
	"time"

	"github.com/stellar/go/ingest/processors/token_transfer"
	"github.com/stellar/go/xdr"
	"github.com/stellar/stellar-rpc/client"
	"github.com/stellar/stellar-rpc/protocol"
)

type EventServer struct {
	event_service.UnimplementedEventServiceServer
	rpcClient *client.Client
	processor *token_transfer.EventsProcessor
}

func NewEventServer(rpcEndpoint string, passphrase string) (*EventServer, error) {
	rpcClient := client.NewClient(rpcEndpoint, nil)
	processor := token_transfer.NewEventsProcessor(passphrase)
	return &EventServer{
		rpcClient: rpcClient,
		processor: processor,
	}, nil
}

// GetEvents implements the gRPC GetEvents method
func (s *EventServer) GetTTPEvents(req *event_service.GetEventsRequest, stream event_service.EventService_GetTTPEventsServer) error {
	ctx := stream.Context()

	// Create initial GetLedgers request
	getLedgersReq := protocol.GetLedgersRequest{
		StartLedger: uint32(req.StartLedger),
	}

	// If EndLedger < StartLedger, do infinite long polling
	if req.EndLedger < req.StartLedger {
		getLedgersReq.Pagination = &protocol.LedgerPaginationOptions{
			Limit: 2,
		}
		for {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				err := ctx.Err()
				log.Printf("Context cancelled in GetTTPEvents: %v", err)
				return err
			default:
			}

			// Get ledgers from RPC
			resp, err := s.rpcClient.GetLedgers(ctx, getLedgersReq)
			if err != nil {
				log.Printf("Error getting ledgers from RPC: %v", err)
				return err
			}

			if len(resp.Ledgers) == 0 {
				// No new ledgers, wait a bit before polling again
				time.Sleep(5 * time.Second)
				continue
			}

			// Process each ledger
			for _, ledgerInfo := range resp.Ledgers {
				// Convert LedgerMetadata to xdr.LedgerCloseMeta
				var ledgerMeta xdr.LedgerCloseMeta
				if err := xdr.SafeUnmarshalBase64(ledgerInfo.LedgerMetadata, &ledgerMeta); err != nil {
					log.Printf("Error unmarshaling ledger metadata: %v: %v", ledgerInfo.LedgerMetadata, err)
					return err
				}

				// Get events from ledger using the token_transfer processor
				events, err := s.processor.EventsFromLedger(ledgerMeta)
				if err != nil {
					log.Printf("Error processing events from ledger %d: %v", ledgerInfo.Sequence, err)
					return err
				}

				// Stream each event
				for i := range events {
					if err := stream.Send(events[i]); err != nil {
						log.Printf("Error sending event to stream: %v", err)
						return err
					}
				}
			}

			// Update cursor for next request
			getLedgersReq.Pagination.Cursor = resp.Cursor
		}
	} else {
		// For finite range, we'll let the RPC server handle pagination
		// since we know the end ledger
		getLedgersReq.StartLedger = uint32(req.EndLedger)
		getLedgersReq.Pagination = &protocol.LedgerPaginationOptions{
			Limit: uint(req.EndLedger - req.StartLedger),
		}

		// Get ledgers from RPC
		resp, err := s.rpcClient.GetLedgers(ctx, getLedgersReq)
		if err != nil {
			log.Printf("Error getting ledgers from RPC: %v", err)
			return err
		}

		// Process each ledger
		for _, ledgerInfo := range resp.Ledgers {
			// Convert LedgerMetadata to xdr.LedgerCloseMeta
			var ledgerMeta xdr.LedgerCloseMeta
			
			if err := xdr.SafeUnmarshalBase64(ledgerInfo.LedgerMetadata, &ledgerMeta); err != nil {
				log.Printf("Error unmarshaling ledger metadata: %v: %v", ledgerInfo.LedgerMetadata, err)
				return err
			}

			// Get events from ledger using the token_transfer processor
			events, err := s.processor.EventsFromLedger(ledgerMeta)
			if err != nil {
				log.Printf("Error processing events from ledger %d: %v", ledgerInfo.Sequence, err)
				return err
			}

			// Stream each event
			for i := range events {
				if err := stream.Send(events[i]); err != nil {
					log.Printf("Error sending event to stream: %v", err)
					return err
				}
			}
		}
	}

	return nil
}
