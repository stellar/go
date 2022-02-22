package index

type Store interface {
	NextActive(index string, afterCheckpoint uint32) (uint32, error)
	AddParticipantsToIndexes(checkpoint uint32, indexFormat string, participants []string) error
	Flush() error
}
