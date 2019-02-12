package horizon

func (op *OperationCallBuilder) ForId(operationId string) {

	op.addEndpoint("/operations/" + operationId)
}
