package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

var paths []string = []string{
	"/payments?order=desc&limit=200&include_failed=true&cursor=97761976872132608",
	"/payments?order=desc&limit=200&include_failed=false&cursor=97761976872132608",
	"/payments?order=desc&limit=200&cursor=97761976872132608",

	"/operations?order=desc&limit=200&include_failed=true&cursor=97761976872132608",
	"/operations?order=desc&limit=200&include_failed=false&cursor=97761976872132608",
	"/operations?order=desc&limit=200&cursor=97761976872132608",

	"/accounts/GBXY56UJJH4MSEHTP35VX7B5JSLKD72XWLXK4JROQBYRASOXNDIMIL2D/payments?order=desc&limit=200&include_failed=true&cursor=97761976872132608",
	"/accounts/GBXY56UJJH4MSEHTP35VX7B5JSLKD72XWLXK4JROQBYRASOXNDIMIL2D/operations?order=desc&limit=200&include_failed=true&cursor=97761976872132608",
	"/accounts/GBXY56UJJH4MSEHTP35VX7B5JSLKD72XWLXK4JROQBYRASOXNDIMIL2D/transactions?order=desc&limit=200&include_failed=true&cursor=97761976872132608",

	"/accounts/GBXY56UJJH4MSEHTP35VX7B5JSLKD72XWLXK4JROQBYRASOXNDIMIL2D/payments?order=desc&limit=2&cursor=97761976872132608",
	"/accounts/GBXY56UJJH4MSEHTP35VX7B5JSLKD72XWLXK4JROQBYRASOXNDIMIL2D/operations?order=desc&limit=2&cursor=97761976872132608",
	"/accounts/GBXY56UJJH4MSEHTP35VX7B5JSLKD72XWLXK4JROQBYRASOXNDIMIL2D/transactions?order=desc&limit=2&cursor=97761976872132608",

	"/accounts/GB3BVRX2D2WBTUYPXOB6S25VTFK36LDCII2AILMMLIJT3SWG66CCS5AK/transactions?order=desc&cursor=97762191620468736",
	"/accounts/GB3BVRX2D2WBTUYPXOB6S25VTFK36LDCII2AILMMLIJT3SWG66CCS5AK/transactions?order=desc&include_failed=true&cursor=97762191620468736",
	"/accounts/GB3BVRX2D2WBTUYPXOB6S25VTFK36LDCII2AILMMLIJT3SWG66CCS5AK/transactions?order=desc&include_failed=false&cursor=97762191620468736",

	"/accounts/GB3BVRX2D2WBTUYPXOB6S25VTFK36LDCII2AILMMLIJT3SWG66CCS5AK/transactions?order=desc&include_failed=true&limit=50&cursor=97696238102654976",
	"/accounts/GB3BVRX2D2WBTUYPXOB6S25VTFK36LDCII2AILMMLIJT3SWG66CCS5AK/transactions?order=desc&include_failed=false&limit=50&cursor=97696238102654976",

	"/ledgers/22764327/operations?limit=200",
	"/ledgers/22764327/operations?limit=200&include_failed=true",
	"/ledgers/22764327/payments?limit=200",
	"/ledgers/22764327/payments?limit=200&include_failed=true",
	"/ledgers/22764327/transactions?limit=200",
	"/ledgers/22764327/transactions?limit=200&include_failed=true",
	"/ledgers/22764327/effects?limit=200",
	"/ledgers/22764327/effects?limit=200&include_failed=true",

	"/ledgers/22762425/operations?limit=200",
	"/ledgers/22762425/operations?limit=200&include_failed=true",
	"/ledgers/22762425/payments?limit=200",
	"/ledgers/22762425/payments?limit=200&include_failed=true",
	"/ledgers/22762425/transactions?limit=200",
	"/ledgers/22762425/transactions?limit=200&include_failed=true",
	"/ledgers/22762425/effects?limit=200",
	"/ledgers/22762425/effects?limit=200&include_failed=true",

	"/payments?limit=200&cursor=97774668500484098",
	"/payments?limit=200&cursor=97774668500484098&include_failed=false",
	"/payments?limit=200&cursor=97774668500484098&include_failed=true",

	"/operations/97774668500484098",
	"/operations/97774689975341057",

	"/operations?limit=200&cursor=97774668500484098",
	"/operations?limit=200&cursor=97774668500484098&include_failed=false",
	"/operations?limit=200&cursor=97774668500484098&include_failed=true",

	"/transactions?limit=200&cursor=97774668500484098",
	"/transactions?limit=200&cursor=97774668500484098&include_failed=false",
	"/transactions?limit=200&cursor=97774668500484098&include_failed=true",

	// failed
	"/transactions/96583f831cefcc5dbab65f918e1818468ed7102f9da1a35159b40c37253ef4a3",
	"/transactions/96583f831cefcc5dbab65f918e1818468ed7102f9da1a35159b40c37253ef4a3/operations",
	"/transactions/96583f831cefcc5dbab65f918e1818468ed7102f9da1a35159b40c37253ef4a3/payments",
	"/transactions/96583f831cefcc5dbab65f918e1818468ed7102f9da1a35159b40c37253ef4a3/effects",

	// succ
	"/transactions/702cbb5fead49b16d64b55a3bfcbabc7d4674a6d5a9e4012caf422ab87db3010",
	"/transactions/702cbb5fead49b16d64b55a3bfcbabc7d4674a6d5a9e4012caf422ab87db3010/operations",
	"/transactions/702cbb5fead49b16d64b55a3bfcbabc7d4674a6d5a9e4012caf422ab87db3010/payments",
	"/transactions/702cbb5fead49b16d64b55a3bfcbabc7d4674a6d5a9e4012caf422ab87db3010/effects",
}

var removeMeta = regexp.MustCompile(`"result_meta_xdr": "(.*)"`)

func main() {
	for _, p := range paths {
		a := getResponse("https://horizon.stellar.org", p)
		fmt.Print(".")
		b := getResponse("https://horizon-dev-pubnet.stellar.org", p)
		fmt.Print(".")

		status := ""
		if a == b {
			status = "ok"
		} else {
			status = "fail"
		}
		fmt.Println(status, p)
		if status == "fail" {
			fmt.Println(a)
			fmt.Println(b)
			return
		}
	}
}

func getResponse(domain, url string) string {
	resp, err := http.Get(domain + url)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		panic(resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	bodyString := string(body)
	// `result_meta_xdr` can differ between core instances (confirmed this with core team)
	bodyString = removeMeta.ReplaceAllString(bodyString, "")
	// Remove Horizon URL from the _links
	bodyString = strings.Replace(bodyString, domain, "", -1)

	return bodyString
}
