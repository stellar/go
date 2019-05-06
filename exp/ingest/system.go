package ingest

func (s *System) Run(session Session) error {
	reducedState := &ReduceStateProcessor{}
	p2 := Processor()
	p3 := Processor()
	p4 := Processor()
	p5 := Processor()
	p6 := Processor()

	pipeline.Node(reducedState).Pipe(
		pipeline.Node(accounts).Pipe(
			pipeline.Node(hotWallet).Pipe(updateRedis),
			pipeline.Node(coldWallet).Pipe(updateRedis),
		),
	)
}
