package utils

type SimpleResponse struct {
	Status string `json:"status"`
	Message string `json:"message"`
}

func NewSimpleResponse(status, message string) *SimpleResponse {
	return &SimpleResponse{status, message}
}