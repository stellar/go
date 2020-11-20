package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/stellar/go/clients/stellartoml"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
)

func main() {
	homeDomains := os.Args[1:]

	w := csv.NewWriter(os.Stdout)
	err := w.Write(
		[]string{"Home Domain", "Success?", "Error from Go SDK's ReadChallengeTx", "Challenge Transaction Returned"},
	)
	if err != nil {
		log.Fatal("error writing csv headers:", err)
	}

	for _, hd := range homeDomains {
		challengeTx, err := check(hd)

		failPassText := "pass"
		if err != nil {
			failPassText = "fail"
		}

		errText := ""
		if err != nil {
			errText = err.Error()
		}

		record := []string{
			hd,
			failPassText,
			errText,
			challengeTx,
		}

		if err = w.Write(record); err != nil {
			log.Fatal("error writing record to csv:", err)
		}
		w.Flush()
	}
}

func check(homeDomain string) (string, error) {
	tomlResp, err := stellartoml.DefaultClient.GetStellarToml(homeDomain)
	if err != nil {
		return "", err
	}
	if tomlResp.WebAuthEndpoint == "" {
		return "", errors.New("SEP-10 not supported")
	}

	queryParams := url.Values{}
	queryParams.Set("account", "GAAZI4TCR3TY5OJHCTJC2A4QSY6CJWJH5IAJTGKIN2ER7LBNVKOCCWN7")
	webAuthResp, err := http.Get(tomlResp.WebAuthEndpoint + "?" + queryParams.Encode())
	if err != nil {
		return "", errors.New("Unable to get SEP-10 challenge")
	}
	webAuthRespMsg := struct {
		Transaction string
	}{}
	tx, err := decodeAndRead(
		webAuthResp,
		webAuthRespMsg,
		tomlResp.SigningKey,
		tomlResp.NetworkPassphrase,
		homeDomain,
	)
	if err != nil {
		return tx, err
	}

	kp, _ := keypair.Random()
	queryParams.Del("account")
	queryParams.Set("account", kp.Address())
	webAuthResp, err = http.Get(tomlResp.WebAuthEndpoint + "?" + queryParams.Encode())
	if err != nil {
		return "", errors.New("Unable to specify account that does not exist")
	}
	tx, err = decodeAndRead(
		webAuthResp,
		webAuthRespMsg,
		tomlResp.SigningKey,
		tomlResp.NetworkPassphrase,
		homeDomain,
	)
	if err != nil {
		return tx, err
	}

	queryParams.Set("home_domain", homeDomain)
	webAuthResp, err = http.Get(tomlResp.WebAuthEndpoint + "?" + queryParams.Encode())
	if err != nil {
		return "", errors.New("Unable to specify home_domain parameter")
	}
	tx, err = decodeAndRead(
		webAuthResp,
		webAuthRespMsg,
		tomlResp.SigningKey,
		tomlResp.NetworkPassphrase,
		homeDomain,
	)
	if err != nil {
		return tx, err
	}

	return tx, nil
}

func decodeAndRead(webAuthResp *http.Response, webAuthRespMsg struct{ Transaction string }, signingKey, passphrase, homeDomain string) (string, error) {
	err := json.NewDecoder(webAuthResp.Body).Decode(&webAuthRespMsg)
	if err != nil {
		return "", errors.New("GET response does not match expectation")
	}
	_, _, _, err = txnbuild.ReadChallengeTx(webAuthRespMsg.Transaction, signingKey, passphrase, []string{homeDomain})
	if err != nil {
		return webAuthRespMsg.Transaction, errors.New("Unable to read challenge: " + err.Error())
	}
	return webAuthRespMsg.Transaction, nil
}
