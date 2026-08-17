package contracts

// Envelope is the common API response shape.
type Envelope struct {
	Data        any    `json:"data"`
	ErrorCode   string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

func OK(data any) Envelope { return Envelope{Data: data} }
func Fail(code, message string) Envelope {
	return Envelope{ErrorCode: code, ErrorMessage: message}
}

