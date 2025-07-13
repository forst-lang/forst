package echo

// EchoRequest: TypeDefShapeExpr({message: String})
type EchoRequest struct {
	message string `json:"message"`
}

// EchoResponse: TypeDefShapeExpr({echo: String, timestamp: Int})
type EchoResponse struct {
	echo      string `json:"echo"`
	timestamp int    `json:"timestamp"`
}

func Echo(input EchoRequest) EchoResponse {
	return EchoResponse{echo: input.message, timestamp: 1234567890}
}
