package horizonclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/manucorporat/sse"
	"github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/support/app"
	"github.com/stellar/go/support/errors"
)

// EffectHandler is a function that is called when a new effect is received
type EffectHandler func(effects.Base)

// BuildUrl creates the endpoint to be queried based on the data in the EffectRequest struct.
// If no data is set, it defaults to the build the URL for all effects
func (er EffectRequest) BuildUrl() (endpoint string, err error) {

	nParams := countParams(er.ForAccount, er.ForLedger, er.ForOperation, er.ForTransaction)

	if nParams > 1 {
		err = errors.New("Invalid request. Too many parameters")
	}

	if err != nil {
		return endpoint, err
	}

	endpoint = "effects"

	if er.ForAccount != "" {
		endpoint = fmt.Sprintf(
			"accounts/%s/effects",
			er.ForAccount,
		)
	}

	if er.ForLedger != "" {
		endpoint = fmt.Sprintf(
			"ledgers/%s/effects",
			er.ForLedger,
		)
	}

	if er.ForOperation != "" {
		endpoint = fmt.Sprintf(
			"operations/%s/effects",
			er.ForOperation,
		)
	}

	if er.ForTransaction != "" {
		endpoint = fmt.Sprintf(
			"transactions/%s/effects",
			er.ForTransaction,
		)
	}

	queryParams := addQueryParams(er.Cursor, er.Limit, er.Order)
	if queryParams != "" {
		endpoint = fmt.Sprintf(
			"%s?%s",
			endpoint,
			queryParams,
		)
	}

	_, err = url.Parse(endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to parse endpoint")
	}

	return endpoint, err
}

// To do: move this from here
func stream(
	ctx context.Context,
	baseURL string,
	cursor *Cursor,
	handler func(data []byte) error,
) error {
	query := url.Values{}
	if cursor != nil {
		query.Set("cursor", string(*cursor))
	}

	client := http.Client{}

	for {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s?%s", baseURL, query.Encode()), nil)
		if err != nil {
			return errors.Wrap(err, "Error creating HTTP request")
		}
		req.Header.Set("Accept", "text/event-stream")
		// to do: confirm name and version
		req.Header.Set("X-Client-Name", "go-stellar-sdk")
		req.Header.Set("X-Client-Version", app.Version())

		// Make sure we don't use c.HTTP that can have Timeout set.
		resp, err := client.Do(req)
		if err != nil {
			return errors.Wrap(err, "Error sending HTTP request")
		}

		// Expected statusCode are 200-299
		if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
			return fmt.Errorf("Got bad HTTP status code %d", resp.StatusCode)
		}
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)

		// Read events one by one. Break this loop when there is no more data to be
		// read from resp.Body (io.EOF).
	Events:
		for {
			// Read until empty line = event delimiter. The perfect solution would be to read
			// as many bytes as possible and forward them to sse.Decode. However this
			// requires much more complicated code.
			// We could also write our own `sse` package that works fine with streams directly
			// (github.com/manucorporat/sse is just using io/ioutils.ReadAll).
			var buffer bytes.Buffer
			nonEmptylinesRead := 0
			for {
				// Check if ctx is not cancelled
				select {
				case <-ctx.Done():
					return nil
				default:
					// Continue
				}

				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF || err == io.ErrUnexpectedEOF {
						// We catch EOF errors to handle two possible situations:
						// - The last line before closing the stream was not empty. This should never
						//   happen in Horizon as it always sends an empty line after each event.
						// - The stream was closed by the server/proxy because the connection was idle.
						//
						// In the former case, that (again) should never happen in Horizon, we need to
						// check if there are any events we need to decode. We do this in the `if`
						// statement below just in case if Horizon behaviour changes in a future.
						//
						// From spec:
						// > Once the end of the file is reached, the user agent must dispatch the
						// > event one final time, as defined below.
						if nonEmptylinesRead == 0 {
							break Events
						}
					} else {
						return errors.Wrap(err, "Error reading line")
					}
				}

				buffer.WriteString(line)

				if strings.TrimRight(line, "\n\r") == "" {
					break
				}

				nonEmptylinesRead++
			}

			events, err := sse.Decode(strings.NewReader(buffer.String()))
			if err != nil {
				return errors.Wrap(err, "Error decoding event")
			}

			// Right now len(events) should always be 1. This loop will be helpful after writing
			// new SSE decoder that can handle io.Reader without using ioutils.ReadAll().
			for _, event := range events {
				if event.Event != "message" {
					continue
				}

				// Update cursor with event ID
				if event.Id != "" {
					query.Set("cursor", event.Id)
				}

				switch data := event.Data.(type) {
				case string:
					err = handler([]byte(data))
					err = errors.Wrap(err, "Handler error")
				case []byte:
					err = handler(data)
					err = errors.Wrap(err, "Handler error")
				default:
					err = errors.New("Invalid event.Data type")
				}
				if err != nil {
					return err
				}
			}
		}
	}
}

func (er EffectRequest) Stream(
	ctx context.Context,
	horizonURL string,
	handler func(interface{}),
) (err error) {

	url := fmt.Sprintf("%s/effects", horizonURL)
	return stream(ctx, url, &er.Cursor, func(data []byte) error {
		var effect effects.Base
		err = json.Unmarshal(data, &effect)
		if err != nil {
			return errors.Wrap(err, "Error unmarshaling data")
		}
		handler(effect)
		return nil
	})
}
