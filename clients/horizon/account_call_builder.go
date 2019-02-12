package horizon

func (ac *AccountCallBuilder) LoadAccount(accountID string) (Account, error) {

	ac.addEndpoint("/accounts/" + accountID)

	resp, err := ac.Call()
	if err != nil {
		return Account{}, err
	}

	// fmt.Print("resp b4 type assert: ", resp)

	// acc, ok := resp.(Account)
	// if !ok {
	// 	return Account{}, errors.New("could not assert type")
	// }

	return resp, nil

}

func (ac *AccountCallBuilder) Call() (acc Account, err error) {

	endpoint, err := ac.buildUrl()
	if err != nil {
		return acc, err
	}

	resp, err := ac.HTTP.Get(endpoint)
	if err != nil {
		return acc, err
	}

	err = decodeResponse(resp, &acc)
	if err != nil {
		return acc, err
	}

	return acc, nil

}
