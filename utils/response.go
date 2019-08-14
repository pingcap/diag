package utils

type SimpleResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func NewSimpleResponse(status, message string) *SimpleResponse {
	return &SimpleResponse{status, message}
}

type PaginationResponse struct {
	Total int         `json:"total"`
	Data  interface{} `json:"data"`
}

func NewPaginationResponse(total int, data interface{}) *PaginationResponse {
	return &PaginationResponse{total, data}
}
