package horizon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stellar/go/xdr"
)

// HomeDomainForAccount returns the home domain for the provided strkey-encoded
// account id.
func (c *Client) HomeDomainForAccount(aid string) (string, error) {
	a, err := c.LoadAccount(aid)
	if err != nil {
		return "", errors.Wrap(err, "load account failed")
	}
	return a.HomeDomain, nil
}

// LoadAccount loads the account state from horizon. err can be either error
// object or horizon.Error object.
func (c *Client) LoadAccount(accountID string) (account Account, err error) {
	resp, err := c.HTTP.Get(c.URL + "/accounts/" + accountID)
	if err != nil {
		return
	}

	err = decodeResponse(resp, &account)
	return
}

// LoadAccountOffers loads the account offers from horizon. err can be either error
// object or horizon.Error object.
func (c *Client) LoadAccountOffers(accountID string, params ...interface{}) (offers OffersPage, err error) {
	query := url.Values{}
	for _, param := range params {
		switch param := param.(type) {
		case Limit:
			query.Add("limit", strconv.Itoa(int(param)))
		case Order:
			query.Add("order", string(param))
		case Cursor:
			query.Add("cursor", string(param))
		default:
			err = fmt.Errorf("Undefined parameter: %+v", param)
			return
		}
	}

	var q string
	if len(query) > 0 {
		q = "?" + query.Encode()
	}

	url := fmt.Sprintf("%s/accounts/%s/offers%s", c.URL, accountID, q)
	resp, err := c.HTTP.Get(url)
	if err != nil {
		return
	}

	err = decodeResponse(resp, &offers)
	return
}

// LoadMemo loads memo for a transaction in Payment
func (c *Client) LoadMemo(p *Payment) (err error) {
	res, err := c.HTTP.Get(p.Links.Transaction.Href)
	if err != nil {
		return errors.Wrap(err, "load transaction failed")
	}
	defer res.Body.Close()
	return json.NewDecoder(res.Body).Decode(&p.Memo)
}

// SequenceForAccount implements build.SequenceProvider
func (c *Client) SequenceForAccount(
	accountID string,
) (xdr.SequenceNumber, error) {

	a, err := c.LoadAccount(accountID)
	if err != nil {
		return 0, errors.Wrap(err, "load account failed")
	}

	seq, err := strconv.ParseUint(a.Sequence, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "parse sequence failed")
	}

	return xdr.SequenceNumber(seq), nil
}

// LoadOrderBook loads order book for given selling and buying assets.
func (c *Client) LoadOrderBook(selling Asset, buying Asset) (orderBook OrderBookSummary, err error) {
	query := url.Values{}

	query.Add("selling_asset_type", selling.Type)
	query.Add("selling_asset_code", selling.Code)
	query.Add("selling_asset_issuer", selling.Issuer)

	query.Add("buying_asset_type", buying.Type)
	query.Add("buying_asset_code", buying.Code)
	query.Add("buying_asset_issuer", buying.Issuer)

	resp, err := c.HTTP.Get(c.URL + "/order_book?" + query.Encode())
	if err != nil {
		return
	}

	err = decodeResponse(resp, &orderBook)
	return
}

func (c *Client) stream(url string, cursor *Cursor, handler func(data []byte) error) (err error) {
	if cursor != nil {
		url += "?cursor=" + string(*cursor)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(splitSSE)

	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}

		ev, err := parseEvent(scanner.Bytes())
		if err != nil {
			return err
		}

		if ev.Event != "message" {
			continue
		}

		switch data := ev.Data.(type) {
		case string:
			err = handler([]byte(data))
		case []byte:
			err = handler(data)
		default:
			err = errors.New("Invalid ev.Data type")
		}
		if err != nil {
			return err
		}
	}

	err = scanner.Err()
	if err == io.ErrUnexpectedEOF {
		return nil
	}
	if err != nil {
		return err
	}

	return nil
}

// StreamLedgers streams incoming ledgers
func (c *Client) StreamLedgers(cursor *Cursor, handler LedgerHandler) (err error) {
	url := fmt.Sprintf("%s/ledgers", c.URL)
	return c.stream(url, cursor, func(data []byte) error {
		var ledger Ledger
		err = json.Unmarshal(data, &ledger)
		if err != nil {
			return errors.Wrap(err, "Error unmarshaling data")
		}
		handler(ledger)
		return nil
	})
}

// StreamPayments streams incoming payments
func (c *Client) StreamPayments(accountID string, cursor *Cursor, handler PaymentHandler) (err error) {
	url := fmt.Sprintf("%s/accounts/%s/payments", c.URL, accountID)
	return c.stream(url, cursor, func(data []byte) error {
		var payment Payment
		err = json.Unmarshal(data, &payment)
		if err != nil {
			return errors.Wrap(err, "Error unmarshaling data")
		}
		handler(payment)
		return nil
	})
}

// StreamTransactions streams incoming transactions
func (c *Client) StreamTransactions(accountID string, cursor *Cursor, handler TransactionHandler) (err error) {
	url := fmt.Sprintf("%s/accounts/%s/transactions", c.URL, accountID)
	return c.stream(url, cursor, func(data []byte) error {
		var transaction Transaction
		err = json.Unmarshal(data, &transaction)
		if err != nil {
			return errors.Wrap(err, "Error unmarshaling data")
		}
		handler(transaction)
		return nil
	})
}

// SubmitTransaction submits a transaction to the network. err can be either error object or horizon.Error object.
func (c *Client) SubmitTransaction(
	transactionEnvelopeXdr string,
) (response TransactionSuccess, err error) {
	v := url.Values{}
	v.Set("tx", transactionEnvelopeXdr)

	resp, err := c.HTTP.PostForm(c.URL+"/transactions", v)
	if err != nil {
		err = errors.Wrap(err, "http post failed")
		return
	}

	err = decodeResponse(resp, &response)
	if err != nil {
		err = errors.Wrap(err, "decode response failed")
		return
	}

	return
}
