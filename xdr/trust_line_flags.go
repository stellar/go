package xdr

// IsAuthorized returns true if issuer has authorized account to perform
// transactions with its credit
func (e TrustLineFlags) IsAuthorized() bool {
	return (e & TrustLineFlagsAuthorizedFlag) != 0
}
