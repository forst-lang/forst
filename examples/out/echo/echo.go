package main

// EchoRequest: TypeDefShapeExpr({message: String})
type EchoRequest struct {
	message string `json:"message"`
}

// T_LbdM2TnbF11: TypeDefShapeExpr({echo: Value(Variable(input.message)), timestamp: Value(1234567890)})
type T_LbdM2TnbF11 struct {
	echo      string `json:"echo"`
	timestamp int    `json:"timestamp"`
}

func Echo(input EchoRequest) T_LbdM2TnbF11 {
	return T_LbdM2TnbF11{echo: input.message, timestamp: 1234567890}
}
