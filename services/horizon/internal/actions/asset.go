package actions

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

// AssetStatsHandler is the action handler for the /asset endpoint
type AssetStatsHandler struct {
}

func (handler AssetStatsHandler) validateAssetParams(code, issuer string, pq db2.PageQuery) error {
	if code != "" {
		if !validAssetCode.MatchString(code) {
			return problem.MakeInvalidFieldProblem(
				"asset_code",
				fmt.Errorf("%s is not a valid asset code", code),
			)
		}
	}

	if issuer != "" {
		if _, err := xdr.AddressToAccountId(issuer); err != nil {
			return problem.MakeInvalidFieldProblem(
				"asset_issuer",
				fmt.Errorf("%s is not a valid asset issuer", issuer),
			)
		}
	}

	if pq.Cursor != "" {
		parts := strings.Split(pq.Cursor, ":")
		if len(parts) != 2 {
			return problem.MakeInvalidFieldProblem(
				"cursor",
				errors.New("cursor must contain exactly one colon"),
			)
		}

		cursorCode, cursorIssuer := parts[0], parts[1]
		if !validAssetCode.MatchString(cursorCode) {
			return problem.MakeInvalidFieldProblem(
				"cursor",
				fmt.Errorf("%s is not a valid asset code", cursorCode),
			)
		}

		if _, err := xdr.AddressToAccountId(cursorIssuer); err != nil {
			return problem.MakeInvalidFieldProblem(
				"cursor",
				fmt.Errorf("%s is not a valid asset issuer", cursorIssuer),
			)
		}
	}

	return nil
}

func (handler AssetStatsHandler) findIssuersForAssets(
	historyQ *history.Q,
	assetStats []history.ExpAssetStat,
) (map[string]history.AccountEntry, error) {
	issuerSet := map[string]bool{}
	issuers := []string{}
	for _, assetStat := range assetStats {
		if issuerSet[assetStat.AssetIssuer] {
			continue
		}
		issuerSet[assetStat.AssetIssuer] = true
		issuers = append(issuers, assetStat.AssetIssuer)
	}

	accountsByID := map[string]history.AccountEntry{}
	accounts, err := historyQ.GetAccountsByIDs(issuers)
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		accountsByID[account.AccountID] = account
		delete(issuerSet, account.AccountID)
	}

	if len(issuerSet) > 0 {
		var issuer string
		for key := range issuerSet {
			issuer = key
			break
		}
		return nil, fmt.Errorf("Account for issuer %s does not exist", issuer)
	}

	return accountsByID, nil
}

// GetResourcePage returns a page of offers.
func (handler AssetStatsHandler) GetResourcePage(
	w HeaderWriter,
	r *http.Request,
) ([]hal.Pageable, error) {
	ctx := r.Context()

	code, err := GetString(r, "asset_code")
	if err != nil {
		return nil, err
	}

	issuer, err := GetString(r, "asset_issuer")
	if err != nil {
		return nil, err
	}

	pq, err := GetPageQuery(r, DisableCursorValidation)
	if err != nil {
		return nil, err
	}

	if err = handler.validateAssetParams(code, issuer, pq); err != nil {
		return nil, err
	}

	historyQ, err := historyQFromRequest(r)
	if err != nil {
		return nil, err
	}

	assetStats, err := historyQ.GetAssetStats(code, issuer, pq)
	if err != nil {
		return nil, err
	}

	issuerAccounts, err := handler.findIssuersForAssets(historyQ, assetStats)
	if err != nil {
		return nil, err
	}

	var response []hal.Pageable
	for _, record := range assetStats {
		var assetStatResponse horizon.AssetStat

		resourceadapter.PopulateExpAssetStat(
			ctx,
			&assetStatResponse,
			record,
			issuerAccounts[record.AssetIssuer],
		)
		response = append(response, assetStatResponse)
	}

	return response, nil
}
