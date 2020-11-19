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
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
)

func main() {
	homeDomains := os.Args[1:]

	w := csv.NewWriter(os.Stdout)
	if err := w.Write([]string{"Home Domain", "Success?", "Error from Go SDK's ReadChallengeTx", "Challenge Transaction Returned"}); err != nil {
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

		if err := w.Write(record); err != nil {
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
		return "", errors.New("sep-10 not supported")
	}
	queryParams := url.Values{}
	queryParams.Set("account", "GAAZI4TCR3TY5OJHCTJC2A4QSY6CJWJH5IAJTGKIN2ER7LBNVKOCCWN7")
	queryParams.Set("home_domain", homeDomain)
	webAuthResp, err := http.Get(tomlResp.WebAuthEndpoint + "?" + queryParams.Encode())
	if err != nil {
		return "", err
	}
	webAuthRespMsg := struct {
		Transaction string
	}{}
	err = json.NewDecoder(webAuthResp.Body).Decode(&webAuthRespMsg)
	if err != nil {
		return "", err
	}
	_, _, _, err = txnbuild.ReadChallengeTx(webAuthRespMsg.Transaction, tomlResp.SigningKey, network.PublicNetworkPassphrase, []string{homeDomain})
	if err != nil {
		return webAuthRespMsg.Transaction, err
	}
	return webAuthRespMsg.Transaction, nil
}
