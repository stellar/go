package listener

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/services/bridge/database"
	supportDB "github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

func (l *PaymentListener) Listen() error {
	l.log = log.DefaultLogger.WithField("service", "PaymentListener")
	l.log.Info("PaymentListener starting...")

	_, err := l.Horizon.LoadAccount(l.ReceivingAccount)
	if err != nil {
		return errors.Wrap(err, "Error loading receiving account")
	}

	cursorValue, err := l.Database.GetListenerLastCursorValue()
	if err != nil {
		l.log.WithField("err", err).Error("Error getting the last cursor")
		return err
	}

	if cursorValue == "" {
		cursorValue = "now"
	}

	cursor := horizon.Cursor(cursorValue)

	ctx := context.Background()
	return l.Horizon.StreamPayments(ctx, l.ReceivingAccount, &cursor, l.onPayment)
}

func (l *PaymentListener) onPayment(payment horizon.Payment) {
	localLog := l.log.WithField("payment", payment)
	localLog.Info("New payment")

	_, err := l.Database.GetReceivedPaymentByOperationID(payment.ID)
	if err == nil {
		localLog.WithField("id", payment.ID).Info("Payment already exists")
		return
	} else if !supportDB.NoRows(err) {
		localLog.WithField("err", err).Error("Error checking if receive payment exists")
		return
	}

	dbPayment := database.ReceivedPayment{
		OperationID: payment.ID,
		ProcessedAt: time.Now(),
		PagingToken: payment.PagingToken,
		Status:      string(PaymentStatusProcessing),
	}

	err = l.Database.InsertReceivedPayment(dbPayment)
	if err != nil {
		localLog.WithField("err", err).Info("Error inserting payment")
		return
	}

	ok, status := l.shouldProcessPayment(payment)
	if !ok {
		dbPayment.Status = string(status)
		l.log.WithField("status", dbPayment.Status).Info("Ignoring payment")
	} else {
		err := l.process(payment)

		if err != nil {
			localLog.WithField("err", err).Error("Payment processed with errors")
			dbPayment.Status = string(PaymentStatusError)
		} else {
			localLog.Info("Payment successfully processed")
			dbPayment.Status = string(PaymentStatusSuccess)
		}
	}

	err = l.Database.UpdateReceivedPaymentStatus(dbPayment.OperationID, string(dbPayment.Status))
	if err != nil {
		localLog.WithField("err", err).Error("Error updating payment")
	}
}

// shouldProcessPayment returns false and status if payment should not be processed
// (ex. asset is different than allowed assets).
func (l *PaymentListener) shouldProcessPayment(payment horizon.Payment) (bool, PaymentStatus) {
	if payment.Type != "payment" && payment.Type != "path_payment" {
		return false, PaymentStatusNotPaymentOperation
	}

	if payment.To != l.ReceivingAccount {
		return false, PaymentStatusNotReceiver
	}

	return true, PaymentStatusProcessing
}

func (l *PaymentListener) process(payment horizon.Payment) error {
	err := l.Horizon.LoadMemo(&payment)
	if err != nil {
		return errors.Wrap(err, "Unable to load transaction memo")
	}

	l.log.WithFields(log.F{"memo": payment.Memo.Value, "type": payment.Memo.Type}).Info("Loaded memo")

	resp, err := l.postForm(
		l.CallbackURL,
		url.Values{
			"id":           {payment.ID},
			"from":         {payment.From},
			"amount":       {payment.Amount},
			"asset_code":   {payment.AssetCode},
			"asset_issuer": {payment.AssetIssuer},
			"memo_type":    {payment.Memo.Type},
			"memo":         {payment.Memo.Value},
		},
	)
	if err != nil {
		return errors.Wrap(err, "Error sending request to receive callback")
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "Error reading receive callback response")
		}

		l.log.WithFields(log.F{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("Error response from receive callback")
		return errors.New("Error response from receive callback")
	}

	return nil
}

func (l *PaymentListener) postForm(url string, form url.Values) (*http.Response, error) {
	strbody := form.Encode()
	req, err := http.NewRequest("POST", url, strings.NewReader(strbody))
	if err != nil {
		return nil, errors.Wrap(err, "configure http request failed")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := l.HTTPClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "http request errored")
	}

	return resp, nil
}
