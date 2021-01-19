package xdr

func (entry *ClaimableBalanceEntry) Flags() Uint32 {
	switch entry.Ext.V {
	case 1:
		return entry.Ext.V1.Flags
	}
	return 0
}
