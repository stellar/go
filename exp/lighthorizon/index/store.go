package index

type Store interface {
	NextActive(account, index string, afterCheckpoint uint32) (uint32, error)
	AddParticipantsToIndexes(checkpoint uint32, index string, participants []string) error
	Flush() error
}
