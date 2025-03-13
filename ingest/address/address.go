package address

func (a *Address) Equals(other *Address) bool {
	if a.AddressType != other.AddressType {
		return false
	}
	if a.StrKey == other.StrKey {
		return true
	}
	return false
}
