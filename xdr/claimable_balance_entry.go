package xdr

func (entry *ClaimableBalanceEntry) Flags() ClaimableBalanceFlags {
	switch entry.Ext.V {
	case 1:
		return ClaimableBalanceFlags(entry.Ext.V1.Flags)
	}
	return 0
}

func NormalizeClaimableBalanceExtension(ext ClaimableBalanceEntryExt) ClaimableBalanceEntryExt {
	normalized := ext
	if ext.V == 1 && ext.V1.Flags == 0 {
		// If the flags equal 0 it is equivalent to Version 0
		normalized = ClaimableBalanceEntryExt{V: 0}
	}
	return normalized
}
