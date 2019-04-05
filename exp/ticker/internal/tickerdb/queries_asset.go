package tickerdb

// InsertAsset inserts a new Asset into the database
func (s *TickerSession) InsertAsset(a *Asset) (err error) {
	tbl := s.GetTable("assets")
	_, err = tbl.Insert(a).IgnoreCols("id").Exec()
	return
}
