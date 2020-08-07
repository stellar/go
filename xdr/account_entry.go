package xdr

func (account *AccountEntry) SignerSummary() map[string]int32 {
	ret := map[string]int32{}

	if account.MasterKeyWeight() > 0 {
		ret[account.AccountId.Address()] = int32(account.Thresholds[0])
	}
	for _, signer := range account.Signers {
		ret[signer.Key.Address()] = int32(signer.Weight)
	}

	return ret
}

func (account *AccountEntry) MasterKeyWeight() byte {
	return account.Thresholds.MasterKeyWeight()
}

func (account *AccountEntry) ThresholdLow() byte {
	return account.Thresholds.ThresholdLow()
}

func (account *AccountEntry) ThresholdMedium() byte {
	return account.Thresholds.ThresholdMedium()
}

func (account *AccountEntry) ThresholdHigh() byte {
	return account.Thresholds.ThresholdHigh()
}

// Liabilities returns AccountEntry's liabilities
func (account *AccountEntry) Liabilities() Liabilities {
	var liabilities Liabilities
	if account.Ext.V1 != nil {
		liabilities = account.Ext.V1.Liabilities
	}
	return liabilities
}
