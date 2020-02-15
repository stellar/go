package account

import "sync"

func NewMemoryStore() Store {
	return &memoryStore{
		accounts: map[string]Account{},
	}
}

type memoryStore struct {
	accountsMu sync.Mutex
	accounts   map[string]Account
}

func (ms *memoryStore) Add(a Account) error {
	ms.accountsMu.Lock()
	defer ms.accountsMu.Unlock()

	if _, ok := ms.accounts[a.Address]; ok {
		return ErrAlreadyExists
	}

	ms.accounts[a.Address] = a
	return nil
}

func (ms *memoryStore) Delete(address string) error {
	ms.accountsMu.Lock()
	defer ms.accountsMu.Unlock()

	delete(ms.accounts, address)
	return nil
}

func (ms *memoryStore) Get(address string) (Account, error) {
	ms.accountsMu.Lock()
	defer ms.accountsMu.Unlock()

	a, ok := ms.accounts[address]
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
		if address == a.OwnerIdentities.Address ||
			address == a.OtherIdentities.Address {
			accounts = append(accounts, a)
		}
	}
	return accounts, nil
}

func (ms *memoryStore) FindWithIdentityPhoneNumber(phoneNumber string) ([]Account, error) {
	ms.accountsMu.Lock()
	defer ms.accountsMu.Unlock()

	accounts := []Account{}
	for _, a := range ms.accounts {
		if phoneNumber == a.OwnerIdentities.PhoneNumber ||
			phoneNumber == a.OtherIdentities.PhoneNumber {
			accounts = append(accounts, a)
		}
	}
	return accounts, nil
}

func (ms *memoryStore) FindWithIdentityEmail(email string) ([]Account, error) {
	ms.accountsMu.Lock()
	defer ms.accountsMu.Unlock()

	accounts := []Account{}
	for _, a := range ms.accounts {
		if email == a.OwnerIdentities.Email ||
			email == a.OtherIdentities.Email {
			accounts = append(accounts, a)
		}
	}
	return accounts, nil
}
