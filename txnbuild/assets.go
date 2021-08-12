package txnbuild

// Assets represents a list of Stellar assets. Importantly, it is sortable.
type Assets []Asset

func (s Assets) Len() int      { return len(s) }
func (s Assets) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s Assets) Less(i, j int) bool {
	iType, err := s[i].GetType()
	if err != nil {
		// TODO: Figure out what to do here... :/ for now, just sort these first?
		return true
	}
	jType, err := s[j].GetType()
	if err != nil {
		// TODO: Figure out what to do here... :/ for now, just sort these first?
		return false
	}

	if int32(iType) < int32(jType) {
		return true
	} else if int32(jType) < int32(iType) {
		return false
	}

	if s[i].GetCode() < s[j].GetCode() {
		return true
	} else if s[j].GetCode() < s[i].GetCode() {
		return false
	}

	return s[i].GetIssuer() < s[j].GetIssuer()
}
