package horizon

func (op *OperationCallBuilder) ForId(operationId string) {

	op.addEndpoint("/operations/" + operationId)
}

func (opc *OperationCallBuilder) Call() (op interface{}, err error) {

	endpoint, err := opc.buildUrl()
	if err != nil {
		return op, err
	}

	resp, err := opc.HTTP.Get(endpoint)
	if err != nil {
		return op, err
	}

	err = decodeResponse(resp, &op)
	if err != nil {
		return op, err
	}

	return op, nil

}
