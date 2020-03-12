package account

import "sync"

func NewMemoryStore() Store {
	return &memoryStore{
		accounts:           []Account{},
		accountsAddressMap: map[string]Account{},
	}
}

type memoryStore struct {
	accountsMu         sync.Mutex
	accounts           []Account
	accountsAddressMap map[string]Account
}

func (ms *memoryStore) Add(a Account) error {
	ms.accountsMu.Lock()
	defer ms.accountsMu.Unlock()

	if _, ok := ms.accountsAddressMap[a.Address]; ok {
		return ErrAlreadyExists
	}

	ms.accounts = append(ms.accounts, a)
	ms.accountsAddressMap[a.Address] = a
	return nil
}

func (ms *memoryStore) Delete(address string) error {
	ms.accountsMu.Lock()
	defer ms.accountsMu.Unlock()

	delete(ms.accountsAddressMap, address)
	return nil
}

func (ms *memoryStore) Get(address string) (Account, error) {
	ms.accountsMu.Lock()
	defer ms.accountsMu.Unlock()

	a, ok := ms.accountsAddressMap[address]
	if !ok {
		return Account{}, ErrNotFound
	}

	return a, nil
}

func (ms *memoryStore) FindWithIdentityAddress(address string) ([]Account, error) {
	ms.accountsMu.Lock()
	defer ms.accountsMu.Unlock()

	accounts := []Account{}
	for _, a := range ms.accounts {
		for _, i := range a.Identities {
			for _, m := range i.AuthMethods {
				if m.Type == AuthMethodTypeAddress && m.Value == address {
					accounts = append(accounts, a)
				}
			}
		}
	}
	return accounts, nil
}

func (ms *memoryStore) FindWithIdentityPhoneNumber(phoneNumber string) ([]Account, error) {
	ms.accountsMu.Lock()
	defer ms.accountsMu.Unlock()

	accounts := []Account{}
	for _, a := range ms.accounts {
		for _, i := range a.Identities {
			for _, m := range i.AuthMethods {
				if m.Type == AuthMethodTypePhoneNumber && m.Value == phoneNumber {
					accounts = append(accounts, a)
				}
			}
		}
	}
	return accounts, nil
}

func (ms *memoryStore) FindWithIdentityEmail(email string) ([]Account, error) {
	ms.accountsMu.Lock()
	defer ms.accountsMu.Unlock()

	accounts := []Account{}
	for _, a := range ms.accounts {
		for _, i := range a.Identities {
			for _, m := range i.AuthMethods {
				if m.Type == AuthMethodTypeEmail && m.Value == email {
					accounts = append(accounts, a)
				}
			}
		}
	}
	return accounts, nil
}
